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

// DecodeError はデコード失敗の詳細を保持する。
type DecodeError struct {
	Index int
	Err   error
}

func (e DecodeError) Error() string {
	return fmt.Sprintf("QR #%d: %v", e.Index, e.Err)
}

// DecodeFile はファイルパスを受け取り、含まれるQRコードのUTF-8文字列リストを返す。
// 戻り値の []DecodeError には、個々のQRのデコード失敗詳細が含まれる。
func DecodeFile(path string) ([]string, []DecodeError) {
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
func decodeImage(path string) ([]string, []DecodeError) {
	f, err := os.Open(path)
	if err != nil {
		return nil, []DecodeError{{Err: fmt.Errorf("open image: %w", err)}}
	}
	defer func() { _ = f.Close() }()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, []DecodeError{{Err: fmt.Errorf("decode image: %w", err)}}
	}

	var bestTexts []string
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
		texts := decodeQRsIndividual(bmp)
		total := 0
		for _, t := range texts {
			total += len(t)
		}
		if total > bestTotal {
			bestTotal = total
			bestTexts = texts
		}
	}

	return bestTexts, nil
}

// decodeQRsIndividual はビットマップ内の各QRコードを個別にデコードして返す。
// QR Structured Append (SA) 形式のQRコードはシーケンス番号順に並べ替えるが、
// SAマージは行わず各物理QRコードのテキストを個別に返す。
func decodeQRsIndividual(bmp *gozxing.BinaryBitmap) []string {
	matrix, err := bmp.GetBlackMatrix()
	if err != nil {
		return nil
	}

	detectorResults, err := gozxingdetector.NewMultiDetector(matrix).DetectMulti(nil)
	if err != nil || len(detectorResults) == 0 {
		return nil
	}

	dec := gozxingqr.NewQRCodeReader().(*gozxingqr.QRCodeReader).GetDecoder()

	type qrEntry struct {
		text   string
		seqNum int
		isSA   bool
		pointY float64
		pointX float64
	}

	var saEntries []qrEntry
	var nonSAEntries []qrEntry

	for _, det := range detectorResults {
		decoderResult, err := dec.Decode(det.GetBits(), nil)
		if err != nil {
			continue
		}
		pts := det.GetPoints()
		var py, px float64
		if len(pts) > 0 {
			py = float64(pts[0].GetY())
			px = float64(pts[0].GetX())
		}
		if decoderResult.HasStructuredAppend() {
			saEntries = append(saEntries, qrEntry{
				text:   decoderResult.GetText(),
				seqNum: decoderResult.GetStructuredAppendSequenceNumber(),
				isSA:   true,
				pointY: py,
				pointX: px,
			})
		} else {
			nonSAEntries = append(nonSAEntries, qrEntry{
				text:   decoderResult.GetText(),
				pointY: py,
				pointX: px,
			})
		}
	}

	sort.Slice(saEntries, func(i, j int) bool {
		return saEntries[i].seqNum < saEntries[j].seqNum
	})
	sort.Slice(nonSAEntries, func(i, j int) bool {
		if absF64(nonSAEntries[i].pointY-nonSAEntries[j].pointY) > 10 {
			return nonSAEntries[i].pointY < nonSAEntries[j].pointY
		}
		return nonSAEntries[i].pointX < nonSAEntries[j].pointX
	})

	texts := make([]string, 0, len(saEntries)+len(nonSAEntries))
	for _, e := range saEntries {
		texts = append(texts, e.text)
	}
	for _, e := range nonSAEntries {
		texts = append(texts, e.text)
	}
	return texts
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
func decodePDF(path string) ([]string, []DecodeError) {
	f, err := os.Open(path)
	if err != nil {
		return nil, []DecodeError{{Err: fmt.Errorf("open pdf: %w", err)}}
	}
	defer func() { _ = f.Close() }()

	pageImages, err := pdfapi.ExtractImagesRaw(f, nil, nil)
	if err != nil {
		return nil, []DecodeError{{Err: fmt.Errorf("extract images from pdf: %w", err)}}
	}

	var allTexts []string
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
				allTexts = append(allTexts, r.GetText())
			}
		}
	}
	return allTexts, allErrs
}

