package decoder

import (
	"bytes"
	"testing"
)

// bitWriter はテスト用にビット列を組み立てる。
type bitWriter struct {
	buf  []byte
	nbit int
}

func (w *bitWriter) put(v, n int) {
	for i := n - 1; i >= 0; i-- {
		if w.nbit%8 == 0 {
			w.buf = append(w.buf, 0)
		}
		if v>>i&1 == 1 {
			w.buf[len(w.buf)-1] |= 0x80 >> (w.nbit % 8)
		}
		w.nbit++
	}
}

func TestParseDataBytes(t *testing.T) {
	w := &bitWriter{}
	w.put(0x3, 4) // 連結ヘッダ
	w.put(0x01, 8)
	w.put(0xAB, 8)
	w.put(0x7, 4) // ECI 20 (Shift_JIS)
	w.put(20, 8)
	w.put(0x1, 4) // 数字 "12345"
	w.put(5, 10)
	w.put(123, 10)
	w.put(45, 7)
	w.put(0x2, 4) // 英数字 "AB:"
	w.put(3, 9)
	w.put(10*45+11, 11)
	w.put(44, 6)
	w.put(0x8, 4) // 漢字 "漢"（Shift_JIS 0x8ABF）
	w.put(1, 8)
	v := 0x8ABF - 0x8140
	w.put((v>>8)*0xC0+(v&0xFF), 13)
	w.put(0x4, 4) // バイト 0x0A
	w.put(1, 8)
	w.put(0x0A, 8)
	w.put(0x0, 4)

	got, eci, err := parseDataBytes(w.buf, 1)
	if err != nil {
		t.Fatal(err)
	}
	want := append([]byte("12345AB:"), 0x8A, 0xBF, 0x0A)
	if !bytes.Equal(got, want) {
		t.Errorf("got %x, want %x", got, want)
	}
	if eci != eciShiftJIS {
		t.Errorf("eci = %d, want %d", eci, eciShiftJIS)
	}
}

func TestMergeStructuredAppend_GapSkipsParity(t *testing.T) {
	// 「あい」(82A0 82A2) を 82A0 82 | A2 で分割し、3枚目が欠落
	syms := []*saSymbol{
		{data: []byte{0x82, 0xA0, 0x82}, eci: eciUnspecified, parity: 0xFF},
		{data: []byte{0xA2}, eci: eciUnspecified, parity: 0xFF},
		nil,
	}
	texts, err := mergeStructuredAppend(syms)
	if err != nil {
		t.Errorf("parity must not be checked when a symbol is missing: %v", err)
	}
	if texts[0] != "あい" || texts[1] != "" {
		t.Errorf("texts = %q", texts)
	}
}

func TestParseDataBytes_Errors(t *testing.T) {
	cases := map[string]func(w *bitWriter){
		"numeric overflow": func(w *bitWriter) { w.put(0x1, 4); w.put(3, 10); w.put(1000, 10) },
		"unsupported ECI":  func(w *bitWriter) { w.put(0x7, 4); w.put(3, 8); w.put(0x4, 4); w.put(1, 8); w.put('a', 8) },
		"hanzi mode":       func(w *bitWriter) { w.put(0xD, 4); w.put(1, 4); w.put(1, 8) },
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			w := &bitWriter{}
			build(w)
			if _, _, err := parseDataBytes(w.buf, 1); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestParseDataBytes_Version10CountBits(t *testing.T) {
	w := &bitWriter{}
	w.put(0x4, 4)
	w.put(2, 16) // 型番10以上のバイトモードは16ビット
	w.put('o', 8)
	w.put('k', 8)
	got, _, err := parseDataBytes(w.buf, 10)
	if err != nil || string(got) != "ok" {
		t.Errorf("got %q, %v", got, err)
	}
}

func TestMergeStructuredAppend(t *testing.T) {
	sjis := []byte{0x82, 0xA0, 0x82, 0xA2, 0xB1, 0x82, 0xA4} // あいｱう
	utf := []byte("あいう")
	xor := func(b []byte) int {
		var p byte
		for _, c := range b {
			p ^= c
		}
		return int(p)
	}
	tests := []struct {
		name      string
		data      [][]byte
		parity    []int
		wantTexts []string
		wantErr   bool
	}{
		{"sjis 3 parts mid-char", [][]byte{sjis[:1], sjis[1:5], sjis[5:]}, nil, []string{"あ", "いｱ", "う"}, false},
		{"utf8 split mid-char", [][]byte{utf[:4], utf[4:]}, nil, []string{"あい", "う"}, false},
		{"truncated", [][]byte{sjis[:2], sjis[4:]}, nil, nil, true},
		{"header mismatch", [][]byte{sjis[:2], sjis[2:]}, []int{xor(sjis), xor(sjis) ^ 1}, nil, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			full := sjis
			if tc.name == "utf8 split mid-char" {
				full = utf
			}
			syms := make([]*saSymbol, len(tc.data))
			for i, d := range tc.data {
				p := xor(full)
				if tc.parity != nil {
					p = tc.parity[i]
				}
				syms[i] = &saSymbol{data: d, eci: eciUnspecified, parity: p}
			}
			texts, err := mergeStructuredAppend(syms)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantTexts != nil {
				for i, w := range tc.wantTexts {
					if texts[i] != w {
						t.Errorf("texts[%d] = %q, want %q", i, texts[i], w)
					}
				}
			}
		})
	}
}

func TestLooksUTF8(t *testing.T) {
	u := []byte("あいう")
	for _, b := range [][]byte{u, u[1:], u[:len(u)-1], u[2 : len(u)-2]} {
		if !looksUTF8(b) {
			t.Errorf("looksUTF8(%x) = false", b)
		}
	}
	if looksUTF8([]byte{0x82, 0xA0, 0x82, 0xA2}) {
		t.Error("Shift_JIS judged as UTF-8")
	}
}
