package decoder

import (
	"fmt"
	"unicode/utf8"

	"github.com/makiuchi-d/gozxing/common"
	"golang.org/x/text/encoding/japanese"
)

// QR Structured Append（分割QR）の処理。
// 規格上、分割QRは各シンボルのデータを「バイト列のまま」連結してから文字コード変換する。
// gozxing の GetText() はシンボルごとに文字列化（文字コード推定）するため、
// 2バイト文字が分割境界をまたぐとシンボル全体が文字化けする。
// ここでは各シンボルのデータコードワードからバイト列を取り出して連結し、
// 連結パリティ（元データ全体のXOR）も照合する。

const (
	eciUnspecified = -1
	eciShiftJIS    = 20
	eciUTF8        = 26
)

const alphanumericChars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ $%*+-./:"

// saSymbol は SA シンボル1枚分のデコード結果を表す。
type saSymbol struct {
	text   string // gozxing の GetText()（バイト列が取り出せなかった場合のフォールバック）
	data   []byte // データ部のバイト列。取り出せなかった場合は nil
	eci    int
	parity int
}

// charCountBits は型番・モードごとの文字数指示子のビット数を返す。
func charCountBits(version int, mode int) int {
	idx := 0
	switch {
	case version >= 27:
		idx = 2
	case version >= 10:
		idx = 1
	}
	switch mode {
	case 0x1: // 数字
		return [3]int{10, 12, 14}[idx]
	case 0x2: // 英数字
		return [3]int{9, 11, 13}[idx]
	case 0x4: // 8ビットバイト
		return [3]int{8, 16, 16}[idx]
	case 0x8: // 漢字
		return [3]int{8, 10, 12}[idx]
	}
	return 0
}

// parseDataBytes はデータコードワードのビット列を解析し、格納されたバイト列と ECI 値を返す。
// 数字・英数字は ASCII、漢字モードは Shift_JIS のバイト列として返す。
// 対応しないモード（中国漢字など）を含む場合はエラーを返す。
func parseDataBytes(codewords []byte, version int) ([]byte, int, error) {
	bits := common.NewBitSource(codewords)
	var out []byte
	eci := eciUnspecified
	read := func(n int) (int, error) {
		if bits.Available() < n {
			return 0, fmt.Errorf("bit stream too short: need %d, available %d", n, bits.Available())
		}
		return bits.ReadBits(n)
	}

	for bits.Available() >= 4 {
		mode, _ := bits.ReadBits(4)
		switch mode {
		case 0x0: // 終端パターン
			return out, eci, checkECI(eci)
		case 0x3: // 連結（シーケンス番号8 + パリティ8）
			if _, err := read(16); err != nil {
				return nil, eci, err
			}
		case 0x5: // FNC1（第1位置）
		case 0x9: // FNC1（第2位置）
			if _, err := read(8); err != nil {
				return nil, eci, err
			}
		case 0x7: // ECI
			first, err := read(8)
			if err != nil {
				return nil, eci, err
			}
			switch {
			case first&0x80 == 0:
				eci = first & 0x7F
			case first&0xC0 == 0x80:
				rest, err := read(8)
				if err != nil {
					return nil, eci, err
				}
				eci = (first&0x3F)<<8 | rest
			case first&0xE0 == 0xC0:
				rest, err := read(16)
				if err != nil {
					return nil, eci, err
				}
				eci = (first&0x1F)<<16 | rest
			default:
				return nil, eci, fmt.Errorf("invalid ECI designator: %#x", first)
			}
		case 0x1, 0x2, 0x4, 0x8:
			count, err := read(charCountBits(version, mode))
			if err != nil {
				return nil, eci, err
			}
			out, err = readSegment(read, mode, count, out)
			if err != nil {
				return nil, eci, err
			}
		default:
			return nil, eci, fmt.Errorf("unsupported mode: %#x", mode)
		}
	}
	return out, eci, checkECI(eci)
}

// checkECI は自前で文字コード変換できる ECI かを確認する。
// それ以外（ISO-8859-1 など）はエラーとし、gozxing の GetText() にフォールバックさせる。
func checkECI(eci int) error {
	switch eci {
	case eciUnspecified, eciShiftJIS, eciUTF8:
		return nil
	}
	return fmt.Errorf("unsupported ECI: %d", eci)
}

// readSegment は1セグメント分のデータを読み、バイト列として out に追加する。
func readSegment(read func(int) (int, error), mode, count int, out []byte) ([]byte, error) {
	switch mode {
	case 0x1: // 数字: 3桁=10ビット、2桁=7ビット、1桁=4ビット
		for count > 0 {
			n, width := 3, 10
			switch count {
			case 2:
				n, width = 2, 7
			case 1:
				n, width = 1, 4
			}
			v, err := read(width)
			if err != nil {
				return nil, err
			}
			if v >= [4]int{1, 10, 100, 1000}[n] {
				return nil, fmt.Errorf("invalid numeric value: %d", v)
			}
			out = fmt.Appendf(out, "%0*d", n, v)
			count -= n
		}
	case 0x2: // 英数字: 2文字=11ビット、1文字=6ビット
		for count > 0 {
			if count >= 2 {
				v, err := read(11)
				if err != nil {
					return nil, err
				}
				if v/45 >= len(alphanumericChars) {
					return nil, fmt.Errorf("invalid alphanumeric value: %d", v)
				}
				out = append(out, alphanumericChars[v/45], alphanumericChars[v%45])
				count -= 2
				continue
			}
			v, err := read(6)
			if err != nil {
				return nil, err
			}
			if v >= len(alphanumericChars) {
				return nil, fmt.Errorf("invalid alphanumeric value: %d", v)
			}
			out = append(out, alphanumericChars[v])
			count--
		}
	case 0x4: // 8ビットバイト
		for ; count > 0; count-- {
			v, err := read(8)
			if err != nil {
				return nil, err
			}
			out = append(out, byte(v))
		}
	case 0x8: // 漢字: 13ビットを Shift_JIS の2バイトに戻す
		for ; count > 0; count-- {
			v, err := read(13)
			if err != nil {
				return nil, err
			}
			assembled := (v/0xC0)<<8 | v%0xC0
			if assembled < 0x1F00 {
				assembled += 0x8140
			} else {
				assembled += 0xC140
			}
			out = append(out, byte(assembled>>8), byte(assembled))
		}
	}
	return out, nil
}

// isSJISLeadByte は Shift_JIS の2バイト文字の第1バイトかを返す。
func isSJISLeadByte(b byte) bool {
	return (b >= 0x81 && b <= 0x9F) || (b >= 0xE0 && b <= 0xFC)
}

// looksUTF8 はバイト列が UTF-8 として妥当かを返す。分割境界・欠落・末尾切り捨てで
// 先頭や末尾のマルチバイト文字が途切れていても UTF-8 と判定する。
func looksUTF8(b []byte) bool {
	for i := 0; i < utf8.UTFMax-1 && len(b) > 0 && !utf8.RuneStart(b[0]); i++ {
		b = b[1:]
	}
	for i := len(b) - 1; i >= 0 && i >= len(b)-utf8.UTFMax; i-- {
		if utf8.RuneStart(b[i]) {
			if !utf8.FullRune(b[i:]) {
				b = b[:i]
			}
			break
		}
	}
	return utf8.Valid(b)
}

// nextCharBoundary は pos 以上で最初の文字境界の位置を返す（utf8=false なら Shift_JIS として判定）。
// Shift_JIS は第2バイトと第1バイトの値域が重なるため、欠落シンボルの直後から始まる区間では
// 先頭が前の文字の第2バイトだと判別できず、その区間の文字化けは避けられない。
func nextCharBoundary(b []byte, pos int, isUTF8 bool) int {
	if isUTF8 {
		for pos < len(b) && !utf8.RuneStart(b[pos]) {
			pos++
		}
		return pos
	}
	i := 0
	for i < pos {
		if isSJISLeadByte(b[i]) {
			i += 2
		} else {
			i++
		}
	}
	if i > len(b) {
		return len(b)
	}
	return i
}

// decodeBytes は連結済みバイト列を文字列に変換する。
func decodeBytes(b []byte, isUTF8 bool) string {
	if isUTF8 {
		return string(b)
	}
	s, err := japanese.ShiftJIS.NewDecoder().Bytes(b)
	if err != nil {
		return string(b)
	}
	return string(s)
}

// mergeStructuredAppend は位置順に並んだ SA シンボル（欠落位置は nil）を連結して
// 各シンボルのテキストと、パリティ不一致時のエラーを返す。
// 連続して読めたシンボル同士はバイト列を連結してから文字列化し、
// 分割境界をまたぐ文字は前のシンボル側に含める。
// パリティは全シンボルのバイト列が揃った場合のみ照合する。
func mergeStructuredAppend(symbols []*saSymbol) ([]string, error) {
	texts := make([]string, len(symbols))
	complete := true

	for start := 0; start < len(symbols); {
		if symbols[start] == nil || symbols[start].data == nil {
			complete = false
			if symbols[start] != nil {
				texts[start] = symbols[start].text
			}
			start++
			continue
		}
		// バイト列が取れたシンボルの連続区間 [start, end) をまとめて処理
		end := start
		var joined []byte
		var bounds []int // 各シンボルの終端位置（バイト）
		eci := eciUnspecified
		for end < len(symbols) && symbols[end] != nil && symbols[end].data != nil {
			joined = append(joined, symbols[end].data...)
			bounds = append(bounds, len(joined))
			if eci == eciUnspecified {
				eci = symbols[end].eci
			}
			end++
		}
		isUTF8 := eci == eciUTF8 || (eci != eciShiftJIS && looksUTF8(joined))
		prev := 0
		for i, bound := range bounds {
			cut := max(nextCharBoundary(joined, bound, isUTF8), prev)
			texts[start+i] = decodeBytes(joined[prev:cut], isUTF8)
			prev = cut
		}
		start = end
	}

	if !complete {
		return texts, nil
	}
	var parity byte
	for _, s := range symbols {
		for _, c := range s.data {
			parity ^= c
		}
	}
	expected := symbols[0].parity
	for i, s := range symbols[1:] {
		if s.parity != expected {
			return texts, fmt.Errorf("連結パリティのヘッダ値がシンボル間で不一致（1枚目 0x%02X / %d枚目 0x%02X）", expected, i+2, s.parity)
		}
	}
	if int(parity) != expected {
		return texts, fmt.Errorf("連結パリティ不一致（ヘッダ 0x%02X / 実データ 0x%02X）", expected, parity)
	}
	return texts, nil
}
