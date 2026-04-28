package decoder

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/makiuchi-d/gozxing"
	gozxingmulti "github.com/makiuchi-d/gozxing/multi/qrcode"
	gozxingdetector "github.com/makiuchi-d/gozxing/multi/qrcode/detector"
	gozxingqr "github.com/makiuchi-d/gozxing/qrcode"
	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
	xdraw "golang.org/x/image/draw"
)

// DecodeError はファイル全体のデコード失敗詳細を保持する。
type DecodeError struct {
	Index int
	Err   error
}

func (e DecodeError) Error() string {
	return fmt.Sprintf("QR #%d: %v", e.Index, e.Err)
}

// QRResult は1枚の物理QRコードのデコード結果を表す。
// Err が nil の場合は Text にデコード済みテキストが入る。
// Err が非 nil の場合は読み取り失敗（Text は空）。
type QRResult struct {
	Text string
	Err  error
}

// DecodeFile はファイルパスを受け取り、含まれるQRコードの結果リストを返す。
// QR Structured Append 形式の場合は各物理QRを個別のエントリとして返し、
// 読み取れなかった位置には Err が設定されたエントリを挿入する。
func DecodeFile(path string) ([]QRResult, []DecodeError) {
	kind, err := detectFileType(path)
	if err != nil {
		return nil, []DecodeError{{Index: 0, Err: err}}
	}
	switch kind {
	case "image":
		return decodeImage(path)
	case "pdf":
		return decodePDF(path)
	default:
		return nil, []DecodeError{{Index: 0, Err: fmt.Errorf("unsupported file type: %s", path)}}
	}
}

// detectFileType はマジックバイト優先でファイル種別を返す。
func detectFileType(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	magic := make([]byte, 8)
	n, _ := f.Read(magic)
	magic = magic[:n]

	switch {
	case len(magic) >= 4 && string(magic[:4]) == "%PDF":
		return "pdf", nil
	case len(magic) >= 8 && magic[0] == 0x89 && magic[1] == 'P' && magic[2] == 'N' && magic[3] == 'G':
		return "image", nil
	case len(magic) >= 2 && magic[0] == 0xFF && magic[1] == 0xD8:
		return "image", nil // JPEG
	}

	// フォールバック: 拡張子で判別
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".pdf":
		return "pdf", nil
	case ".png", ".jpg", ".jpeg":
		return "image", nil
	}
	return "", fmt.Errorf("cannot determine file type: %s", path)
}

// rawEntry は1枚のQRコードの生デコード結果（ギャップ補完前）を表す。
type rawEntry struct {
	text   string
	err    error
	seqNum int // SA シーケンス番号（raw値: position<<4 | total-1）。非SA は -1
	pointY float64
	pointX float64
}

// decodeRawEntries はビットマップ内の全QRコードパターンを検出・デコードし、
// SA マージもギャップ補完もせず生エントリをそのまま返す。
func decodeRawEntries(bmp *gozxing.BinaryBitmap) []rawEntry {
	matrix, err := bmp.GetBlackMatrix()
	if err != nil {
		return nil
	}
	detectorResults, err := gozxingdetector.NewMultiDetector(matrix).DetectMulti(nil)
	if err != nil || len(detectorResults) == 0 {
		return nil
	}

	dec := gozxingqr.NewQRCodeReader().(*gozxingqr.QRCodeReader).GetDecoder()
	entries := make([]rawEntry, 0, len(detectorResults))

	for _, det := range detectorResults {
		pts := det.GetPoints()
		var py, px float64
		if len(pts) > 0 {
			py = float64(pts[0].GetY())
			px = float64(pts[0].GetX())
		}
		decoderResult, decErr := dec.Decode(det.GetBits(), nil)
		if decErr != nil {
			entries = append(entries, rawEntry{err: decErr, seqNum: -1, pointY: py, pointX: px})
			continue
		}
		if decoderResult.HasStructuredAppend() {
			entries = append(entries, rawEntry{
				text:   decoderResult.GetText(),
				seqNum: decoderResult.GetStructuredAppendSequenceNumber(),
				pointY: py,
				pointX: px,
			})
		} else {
			entries = append(entries, rawEntry{
				text:   decoderResult.GetText(),
				seqNum: -1,
				pointY: py,
				pointX: px,
			})
		}
	}
	return entries
}

// decodeImage は画像ファイルからQRコードをデコードする。
// 解像度が低い画像（スクリーンショット等）では 1x→2x→3x と段階的にスケールアップし、
// SA QR の各ポジションについて成功したスケールの結果を採用する（ポジション単位リトライ）。
// 全スケール試行後に欠落ポジションを1回ギャップ補完して返す。
func decodeImage(path string) ([]QRResult, []DecodeError) {
	f, err := os.Open(path)
	if err != nil {
		return nil, []DecodeError{{Err: fmt.Errorf("open image: %w", err)}}
	}
	defer func() { _ = f.Close() }()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, []DecodeError{{Err: fmt.Errorf("decode image: %w", err)}}
	}

	b := img.Bounds()
	maxDim := b.Dx()
	if b.Dy() > maxDim {
		maxDim = b.Dy()
	}
	scales := []int{1}
	if maxDim < 3000 {
		scales = []int{1, 2, 3}
	}

	// SA QR: ポジション→テキスト（最初に成功したスケールの結果を採用）
	saByPos := make(map[int]string)
	saTotal := 0

	// 非SA QR: テキストで重複排除しつつ位置情報を保持
	type nonSAEntry struct {
		text   string
		pointY float64
		pointX float64
	}
	nonSASeen := make(map[string]bool)
	var nonSAEntries []nonSAEntry

	for _, scale := range scales {
		target := scaleImage(img, scale)
		bmp, err := gozxing.NewBinaryBitmapFromImage(target)
		if err != nil {
			continue
		}
		for _, e := range decodeRawEntries(bmp) {
			if e.err != nil || e.seqNum < 0 && e.text == "" {
				continue // デコード失敗は個別スケールでは無視（全スケール後にギャップ補完）
			}
			if e.seqNum >= 0 {
				// SA QR: まだ成功していないポジションなら採用
				pos := e.seqNum >> 4
				total := (e.seqNum & 0x0F) + 1
				if saTotal == 0 {
					saTotal = total
				}
				if _, ok := saByPos[pos]; !ok {
					saByPos[pos] = e.text
				}
			} else {
				// 非SA QR: テキストが同じなら同一QRとして重複排除
				if !nonSASeen[e.text] {
					nonSASeen[e.text] = true
					nonSAEntries = append(nonSAEntries, nonSAEntry{
						text:   e.text,
						pointY: e.pointY,
						pointX: e.pointX,
					})
				}
			}
		}
	}

	// 非SA QR を位置順（上→下、左→右）に並べ替え
	sort.Slice(nonSAEntries, func(i, j int) bool {
		if absF64(nonSAEntries[i].pointY-nonSAEntries[j].pointY) > 10 {
			return nonSAEntries[i].pointY < nonSAEntries[j].pointY
		}
		return nonSAEntries[i].pointX < nonSAEntries[j].pointX
	})

	if saTotal > 0 {
		// SA QR: 全ポジションを順に並べ、読み取れなかった位置はエラーエントリを挿入
		results := make([]QRResult, saTotal)
		for pos := 0; pos < saTotal; pos++ {
			if text, ok := saByPos[pos]; ok {
				results[pos] = QRResult{Text: text}
			} else {
				results[pos] = QRResult{Err: fmt.Errorf("QRコードを読み取れませんでした（全スケールで失敗）")}
			}
		}
		return results, nil
	}

	// 非SA QR のみ
	results := make([]QRResult, len(nonSAEntries))
	for i, e := range nonSAEntries {
		results[i] = QRResult{Text: e.text}
	}
	return results, nil
}

// scaleImage は画像を整数倍にスケールアップする（scale=1 で元画像を返す）。
// CatmullRom は QR コードのモジュール境界を保ちつつ滑らかな補間を行い、
// gozxing の二値化精度を高める。
func scaleImage(src image.Image, scale int) image.Image {
	if scale <= 1 {
		return src
	}
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx()*scale, b.Dy()*scale))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	return dst
}

func absF64(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// decodePDF はPDFファイルからQRコードをデコードする。
// pdfcpuのExtractImagesRawでPDF内の埋め込み画像を取得しQRをデコードする。
// ベクター描画されたQRは対象外（埋め込み画像としてQRが含まれるPDFに対応）。
func decodePDF(path string) ([]QRResult, []DecodeError) {
	f, err := os.Open(path)
	if err != nil {
		return nil, []DecodeError{{Err: fmt.Errorf("open pdf: %w", err)}}
	}
	defer func() { _ = f.Close() }()

	pageImages, err := pdfapi.ExtractImagesRaw(f, nil, nil)
	if err != nil {
		return nil, []DecodeError{{Err: fmt.Errorf("extract images from pdf: %w", err)}}
	}

	var allResults []QRResult
	var allErrs []DecodeError
	globalIdx := 0
	for _, pageMap := range pageImages {
		for _, imgEntry := range pageMap {
			globalIdx++
			img, _, err := image.Decode(imgEntry)
			if err != nil {
				allErrs = append(allErrs, DecodeError{Index: globalIdx, Err: fmt.Errorf("decode embedded image: %w", err)})
				continue
			}
			bmp, err := gozxing.NewBinaryBitmapFromImage(img)
			if err != nil {
				continue
			}
			reader := gozxingmulti.NewQRCodeMultiReader()
			results, err := reader.DecodeMultiple(bmp, nil)
			if err != nil {
				continue
			}
			for _, r := range results {
				allResults = append(allResults, QRResult{Text: r.GetText()})
			}
		}
	}
	return allResults, allErrs
}
