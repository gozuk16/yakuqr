// gen-sa-testdata-qr は QR Structured Append（分割QR）のテスト画像を生成する。
// 一般的なQRライブラリは連結ヘッダを任意に書けないため、ビット列を自前で組み立て、
// gozxing の MatrixUtil_buildMatrix でシンボル化する。
// 他社アプリの誤設定（誤り訂正レベル・分割数）による末尾切り捨てなど、
// 意図的に壊したデータを作るためのツール。
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/common/reedsolomon"
	"github.com/makiuchi-d/gozxing/qrcode/decoder"
	"github.com/makiuchi-d/gozxing/qrcode/encoder"
	"golang.org/x/text/encoding/japanese"
)

// symbol は1枚の分割QRの内容を表す。
type symbol struct {
	index     int    // 0始まりの位置
	total     int    // 分割総数
	parity    byte   // 連結パリティ（元データ全体のXOR）
	data      []byte // 実際に格納するバイト列
	declared  int    // 文字数指示子に書く長さ（通常は len(data)）
	ecl       decoder.ErrorCorrectionLevel
	version   int // 0 なら data が収まる最小の型番を自動選択
	truncBits bool
}

func dataCapacity(v *decoder.Version, ecl decoder.ErrorCorrectionLevel) int {
	return v.GetTotalCodewords() - v.GetECBlocksForLevel(ecl).GetTotalECCodewords()
}

func countBits(v int) int {
	if v < 10 {
		return 8
	}
	return 16
}

// headerBits は連結ヘッダ(20) + バイトモード指示子(4) + 文字数指示子のビット数。
func headerBits(v int) int { return 20 + 4 + countBits(v) }

// byteCapacity は型番・誤り訂正レベルで1シンボルに入る最大バイト数を返す。
func byteCapacity(version int, ecl decoder.ErrorCorrectionLevel) int {
	v, _ := decoder.Version_GetVersionForNumber(version)
	return (dataCapacity(v, ecl)*8 - headerBits(version)) / 8
}

func smallestVersion(n int, ecl decoder.ErrorCorrectionLevel) int {
	for ver := 1; ver <= 40; ver++ {
		if byteCapacity(ver, ecl) >= n {
			return ver
		}
	}
	panic("data too large")
}

func buildSymbol(s symbol) (*encoder.ByteMatrix, error) {
	ver := s.version
	if ver == 0 {
		ver = smallestVersion(len(s.data), s.ecl)
	}
	v, err := decoder.Version_GetVersionForNumber(ver)
	if err != nil {
		return nil, err
	}
	capBytes := dataCapacity(v, s.ecl)

	bits := gozxing.NewEmptyBitArray()
	_ = bits.AppendBits(0x3, 4) // Structured Append
	_ = bits.AppendBits(s.index, 4)
	_ = bits.AppendBits(s.total-1, 4)
	_ = bits.AppendBits(int(s.parity), 8)
	_ = bits.AppendBits(0x4, 4) // Byte mode
	_ = bits.AppendBits(s.declared, countBits(ver))
	for _, b := range s.data {
		if bits.GetSize()+8 > capBytes*8 {
			if s.truncBits {
				break // 容量オーバー分はビット列ごと捨てる（文字数指示子は元の長さのまま）
			}
			return nil, fmt.Errorf("symbol %d: data overflows version %d-%s", s.index, ver, s.ecl)
		}
		_ = bits.AppendBits(int(b), 8)
	}
	// 終端パターンとパディング
	for i := 0; i < 4 && bits.GetSize() < capBytes*8; i++ {
		_ = bits.AppendBits(0, 1)
	}
	for bits.GetSize()%8 != 0 {
		_ = bits.AppendBits(0, 1)
	}
	for i := 0; bits.GetSize() < capBytes*8; i++ {
		pad := 0xEC
		if i%2 == 1 {
			pad = 0x11
		}
		_ = bits.AppendBits(pad, 8)
	}

	final, err := interleave(bits, v, s.ecl)
	if err != nil {
		return nil, err
	}
	dim := v.GetDimensionForVersion()
	m := encoder.NewByteMatrix(dim, dim)
	if err := encoder.MatrixUtil_buildMatrix(final, s.ecl, v, 2, m); err != nil {
		return nil, err
	}
	return m, nil
}

// interleave はデータコード語をブロック分割し、RS誤り訂正コード語を付けてインターリーブする。
func interleave(bits *gozxing.BitArray, v *decoder.Version, ecl decoder.ErrorCorrectionLevel) (*gozxing.BitArray, error) {
	ecb := v.GetECBlocksForLevel(ecl)
	ecPer := ecb.GetECCodewordsPerBlock()
	raw := make([]byte, bits.GetSizeInBytes())
	bits.ToBytes(0, raw, 0, len(raw))

	rs := reedsolomon.NewReedSolomonEncoder(reedsolomon.GenericGF_QR_CODE_FIELD_256)
	var dataBlocks, ecBlocks [][]byte
	off := 0
	for _, g := range ecb.GetECBlocks() {
		for i := 0; i < g.GetCount(); i++ {
			d := raw[off : off+g.GetDataCodewords()]
			off += g.GetDataCodewords()
			work := make([]int, len(d)+ecPer)
			for j, b := range d {
				work[j] = int(b)
			}
			if err := rs.Encode(work, ecPer); err != nil {
				return nil, err
			}
			ec := make([]byte, ecPer)
			for j := range ec {
				ec[j] = byte(work[len(d)+j])
			}
			dataBlocks = append(dataBlocks, d)
			ecBlocks = append(ecBlocks, ec)
		}
	}
	out := gozxing.NewEmptyBitArray()
	maxLen := 0
	for _, d := range dataBlocks {
		if len(d) > maxLen {
			maxLen = len(d)
		}
	}
	for i := 0; i < maxLen; i++ {
		for _, d := range dataBlocks {
			if i < len(d) {
				_ = out.AppendBits(int(d[i]), 8)
			}
		}
	}
	for i := 0; i < ecPer; i++ {
		for _, e := range ecBlocks {
			_ = out.AppendBits(int(e[i]), 8)
		}
	}
	return out, nil
}

// render は複数シンボルを横に並べた画像（薬情の印刷面を想定）を返す。
func render(ms []*encoder.ByteMatrix) image.Image {
	const scale, quiet = 6, 4
	w, h := 0, 0
	for _, m := range ms {
		w += (m.GetWidth() + quiet*2) * scale
		if hh := (m.GetHeight() + quiet*2) * scale; hh > h {
			h = hh
		}
	}
	img := image.NewGray(image.Rect(0, 0, w, h))
	for i := range img.Pix {
		img.Pix[i] = 0xFF
	}
	x0 := 0
	for _, m := range ms {
		for y := 0; y < m.GetHeight(); y++ {
			for x := 0; x < m.GetWidth(); x++ {
				if m.Get(x, y) != 1 {
					continue
				}
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						img.SetGray(x0+(x+quiet)*scale+dx, (y+quiet)*scale+dy, color.Gray{Y: 0})
					}
				}
			}
		}
		x0 += (m.GetWidth() + quiet*2) * scale
	}
	return img
}

func parityOf(b []byte) byte {
	var p byte
	for _, c := range b {
		p ^= c
	}
	return p
}

// isLeadByte は Shift_JIS の2バイト文字の第1バイトかを返す。
func isLeadByte(b byte) bool {
	return (b >= 0x81 && b <= 0x9F) || (b >= 0xE0 && b <= 0xFC)
}

// charStarts は Shift_JIS バイト列の各文字の開始位置を返す。
func charStarts(b []byte) map[int]bool {
	starts := map[int]bool{}
	for i := 0; i < len(b); {
		starts[i] = true
		if isLeadByte(b[i]) {
			i += 2
		} else {
			i++
		}
	}
	return starts
}

// midCharCut は target 以上で、2バイト文字の途中（第1バイトの直後）になる最初の位置を返す。
func midCharCut(b []byte, target int) int {
	st := charStarts(b)
	for i := target; i < len(b); i++ {
		if st[i] && isLeadByte(b[i]) {
			return i + 1
		}
	}
	panic("no mid-char cut found")
}

// boundaryCut は target 以上で、2バイト文字の直後（文字境界）になる最初の位置を返す。
func boundaryCut(b []byte, target int) int {
	st := charStarts(b)
	for i := target; i < len(b); i++ {
		if st[i] && isLeadByte(b[i]) {
			return i + 2
		}
	}
	panic("no boundary cut found")
}

type scenario struct {
	name    string
	desc    string
	symbols func(full []byte) []symbol
}

func scenarios() []scenario {
	M, L := decoder.ErrorCorrectionLevel_M, decoder.ErrorCorrectionLevel_L
	return []scenario{
		{
			name: "sa_a_valid_char_straddle",
			desc: "正常: 2分割。分割位置が2バイト文字の途中だが欠落なし（規格上は正しいQR）",
			symbols: func(full []byte) []symbol {
				cut := midCharCut(full, len(full)/2)
				p := parityOf(full)
				return []symbol{
					{index: 0, total: 2, parity: p, data: full[:cut], declared: cut, ecl: M},
					{index: 1, total: 2, parity: p, data: full[cut:], declared: len(full) - cut, ecl: M},
				}
			},
		},
		{
			name: "sa_b_truncated_mid_char",
			desc: "不良: 1枚目の末尾6バイト余りが欠落し、切れ目が2バイト文字の途中",
			symbols: func(full []byte) []symbol {
				half := len(full) / 2
				keep := midCharCut(full, half-8)
				p := parityOf(full)
				return []symbol{
					{index: 0, total: 2, parity: p, data: full[:keep], declared: keep, ecl: M},
					{index: 1, total: 2, parity: p, data: full[half:], declared: len(full) - half, ecl: M},
				}
			},
		},
		{
			name: "sa_c_truncated_on_boundary",
			desc: "不良: 1枚目の末尾が欠落、切れ目が文字境界（文字化けしない静かな欠落）",
			symbols: func(full []byte) []symbol {
				half := len(full) / 2
				keep := boundaryCut(full, half-8)
				p := parityOf(full)
				return []symbol{
					{index: 0, total: 2, parity: p, data: full[:keep], declared: keep, ecl: M},
					{index: 1, total: 2, parity: p, data: full[half:], declared: len(full) - half, ecl: M},
				}
			},
		},
		{
			name: "sa_d_overflow_declared_len",
			desc: "不良: 文字数指示子は元の長さのまま、容量を超えたビット列だけが切れている",
			symbols: func(full []byte) []symbol {
				half := len(full) / 2
				p := parityOf(full)
				// 誤り訂正 L 前提の型番に M で詰め込む
				ver := smallestVersion(half, L)
				return []symbol{
					{index: 0, total: 2, parity: p, data: full[:half], declared: half, ecl: M, version: ver, truncBits: true},
					{index: 1, total: 2, parity: p, data: full[half:], declared: len(full) - half, ecl: M, version: smallestVersion(len(full)-half, M)},
				}
			},
		},
		{
			name: "sa_e_misconfigured_ecl",
			desc: "不良（事例の再現）: 誤り訂正 L 前提で分割数・型番を決め、M で出力したため各QRの末尾が容量オーバーで切り捨て",
			symbols: func(full []byte) []symbol {
				const n = 2
				p := parityOf(full)
				chunk := (len(full) + n - 1) / n
				ver := smallestVersion(chunk, L)
				capM := byteCapacity(ver, M)
				var ss []symbol
				for i := 0; i < n; i++ {
					start := i * chunk
					end := start + chunk
					if end > len(full) {
						end = len(full)
					}
					part := full[start:end]
					if len(part) > capM {
						part = part[:capM]
					}
					ss = append(ss, symbol{index: i, total: n, parity: p, data: part, declared: len(part), ecl: M, version: ver})
				}
				return ss
			},
		},
	}
}

func main() {
	cwd, _ := os.Getwd()
	src := filepath.Join(cwd, "testdata", "ver4_single.txt")
	dst := filepath.Join(cwd, "testdata", "generated", "sa")
	if err := os.MkdirAll(dst, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	content, err := os.ReadFile(src)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	full, err := japanese.ShiftJIS.NewEncoder().Bytes(content)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, sc := range scenarios() {
		var ms []*encoder.ByteMatrix
		for _, s := range sc.symbols(full) {
			m, err := buildSymbol(s)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", sc.name, err)
				os.Exit(1)
			}
			ms = append(ms, m)
			fmt.Printf("  %s #%d/%d bytes=%d declared=%d\n", sc.name, s.index+1, s.total, len(s.data), s.declared)
		}
		out := filepath.Join(dst, sc.name+".png")
		f, err := os.Create(out)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := png.Encode(f, render(ms)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		_ = f.Close()
		fmt.Printf("generated: %s  # %s\n", out, sc.desc)
	}
}
