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

// decodeImage は画像ファイルからQRコードをデコードする。
// 解像度が低い画像（スクリーンショット等）ではQRコードを見落とすことがあるため、
// 1x→2x→3x と段階的にスケールアップし、最もテキスト量が多い結果を採用する。
// QR Structured Append (SA) 形式の場合、各物理QRコードを個別に返す（SAマージしない）。
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

	var bestResults []QRResult
	bestTotal := 0

	// 元画像が既に十分大きければスケールアップは不要。
	// スクリーンショット等の小さい画像（最長辺<3000px）は段階的にスケールアップし
	// 最もテキスト量が多い結果を採用する。
	b := img.Bounds()
	maxDim := b.Dx()
	if b.Dy() > maxDim {
		maxDim = b.Dy()
	}
	scales := []int{1}
	if maxDim < 3000 {
		scales = []int{1, 2, 3}
	}

	for _, scale := range scales {
		target := scaleImage(img, scale)
		bmp, err := gozxing.NewBinaryBitmapFromImage(target)
		if err != nil {
			continue
		}
		results := decodeQRsIndividual(bmp)
		total := 0
		for _, r := range results {
			total += len(r.Text)
		}
		if total > bestTotal {
			bestTotal = total
			bestResults = results
		}
	}

	return bestResults, nil
}

// decodeQRsIndividual はビットマップ内の各QRコードを個別にデコードして返す。
// QR Structured Append (SA) 形式のQRコードはシーケンス番号順に並べ替え、
// 欠落した位置には Err を持つエントリを挿入する。
// SAマージは行わず各物理QRコードのテキストを個別に返す。
func decodeQRsIndividual(bmp *gozxing.BinaryBitmap) []QRResult {
	matrix, err := bmp.GetBlackMatrix()
	if err != nil {
		return nil
	}

	detectorResults, err := gozxingdetector.NewMultiDetector(matrix).DetectMulti(nil)
	if err != nil || len(detectorResults) == 0 {
		return nil
	}

	dec := gozxingqr.NewQRCodeReader().(*gozxingqr.QRCodeReader).GetDecoder()

	type entry struct {
		text   string
		err    error
		seqNum int    // SA シーケンス番号（raw 値）。非SA は -1
		pointY float64
		pointX float64
	}

	var saEntries []entry
	var nonSAEntries []entry
	var failedEntries []entry // デコード失敗（SA かどうか不明）

	for _, det := range detectorResults {
		pts := det.GetPoints()
		var py, px float64
		if len(pts) > 0 {
			py = float64(pts[0].GetY())
			px = float64(pts[0].GetX())
		}

		decoderResult, decErr := dec.Decode(det.GetBits(), nil)
		if decErr != nil {
			failedEntries = append(failedEntries, entry{err: decErr, seqNum: -1, pointY: py, pointX: px})
			continue
		}

		if decoderResult.HasStructuredAppend() {
			saEntries = append(saEntries, entry{
				text:   decoderResult.GetText(),
				seqNum: decoderResult.GetStructuredAppendSequenceNumber(),
				pointY: py,
				pointX: px,
			})
		} else {
			nonSAEntries = append(nonSAEntries, entry{
				text:   decoderResult.GetText(),
				seqNum: -1,
				pointY: py,
				pointX: px,
			})
		}
	}

	// 非SA QR を位置順（上→下、左→右）に並べ替え
	sort.Slice(nonSAEntries, func(i, j int) bool {
		if absF64(nonSAEntries[i].pointY-nonSAEntries[j].pointY) > 10 {
			return nonSAEntries[i].pointY < nonSAEntries[j].pointY
		}
		return nonSAEntries[i].pointX < nonSAEntries[j].pointX
	})

	if len(saEntries) == 0 {
		// SA QR なし: 非SA + 失敗のみ返す
		results := make([]QRResult, 0, len(nonSAEntries)+len(failedEntries))
		for _, e := range nonSAEntries {
			results = append(results, QRResult{Text: e.text})
		}
		for _, e := range failedEntries {
			results = append(results, QRResult{Err: e.err})
		}
		return results
	}

	// SA QR をシーケンス番号順に並べ替え
	sort.Slice(saEntries, func(i, j int) bool {
		return saEntries[i].seqNum < saEntries[j].seqNum
	})

	// QR 規格: seqNum = (position << 4) | (total - 1)
	total := (saEntries[0].seqNum & 0x0F) + 1
	byPos := make(map[int]entry, len(saEntries))
	for _, e := range saEntries {
		pos := e.seqNum >> 4
		byPos[pos] = e
	}

	// 欠落位置を補完しながら結果リストを構築
	results := make([]QRResult, 0, total+len(nonSAEntries))
	for pos := 0; pos < total; pos++ {
		if e, ok := byPos[pos]; ok {
			results = append(results, QRResult{Text: e.text})
		} else {
			// 欠落: デコード失敗エントリがあれば割り当て、なければ未検出
			var gapErr error
			if len(failedEntries) > 0 {
				gapErr = failedEntries[0].err
				failedEntries = failedEntries[1:]
			} else {
				gapErr = fmt.Errorf("QRコードを検出できませんでした")
			}
			results = append(results, QRResult{Err: gapErr})
		}
	}

	// 残余の失敗エントリ（位置不明）
	for _, e := range failedEntries {
		results = append(results, QRResult{Err: e.err})
	}

	// 非SA QR を末尾に追加
	for _, e := range nonSAEntries {
		results = append(results, QRResult{Text: e.text})
	}

	return results
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
