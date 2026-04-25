package decoder

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/makiuchi-d/gozxing"
	gozxingmulti "github.com/makiuchi-d/gozxing/multi/qrcode"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
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
	defer f.Close()

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
func decodeImage(path string) ([]string, []DecodeError) {
	f, err := os.Open(path)
	if err != nil {
		return nil, []DecodeError{{Err: fmt.Errorf("open image: %w", err)}}
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, []DecodeError{{Err: fmt.Errorf("decode image: %w", err)}}
	}

	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return nil, []DecodeError{{Err: fmt.Errorf("bitmap: %w", err)}}
	}

	reader := gozxingmulti.NewQRCodeMultiReader()
	results, err := reader.DecodeMultiple(bmp, nil)
	if err != nil {
		return nil, []DecodeError{{Err: fmt.Errorf("qr decode: %w", err)}}
	}

	// 座標順（上→下、左→右）でソート
	sort.Slice(results, func(i, j int) bool {
		pi := results[i].GetResultPoints()
		pj := results[j].GetResultPoints()
		if len(pi) == 0 || len(pj) == 0 {
			return false
		}
		yi := pi[0].GetY()
		yj := pj[0].GetY()
		if absF64(yi-yj) > 10 {
			return yi < yj
		}
		return pi[0].GetX() < pj[0].GetX()
	})

	var texts []string
	var decErrs []DecodeError
	for i, r := range results {
		text, err := toUTF8([]byte(r.GetText()))
		if err != nil {
			decErrs = append(decErrs, DecodeError{Index: i + 1, Err: err})
			continue
		}
		texts = append(texts, text)
	}
	return texts, decErrs
}

func absF64(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// decodePDF はPDFファイルからQRコードをデコードする（Task 4で実装）。
func decodePDF(path string) ([]string, []DecodeError) {
	return nil, []DecodeError{{Err: fmt.Errorf("decodePDF: not yet implemented")}}
}

// toUTF8 はShift_JISバイト列をUTF-8文字列に変換する。
func toUTF8(b []byte) (string, error) {
	dec := japanese.ShiftJIS.NewDecoder()
	out, _, err := transform.Bytes(dec, b)
	if err != nil {
		return "", fmt.Errorf("ShiftJIS->UTF8: %w", err)
	}
	return string(out), nil
}
