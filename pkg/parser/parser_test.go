package parser_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gozuk16/yakuqr/pkg/parser"
)

func readTestdata(name string) string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "testdata")
	b, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		panic(err)
	}
	return string(b)
}

func TestParse_SingleQR_Version(t *testing.T) {
	raw := readTestdata("ver4_single.txt")
	p, _ := parser.Parse([]string{raw})
	if p.Version != parser.Version4 {
		t.Errorf("expected Version4, got %v", p.Version)
	}
}

func TestParse_SingleQR_Records(t *testing.T) {
	raw := readTestdata("ver4_single.txt")
	p, _ := parser.Parse([]string{raw})
	if len(p.Records) == 0 {
		t.Fatal("expected records")
	}
	// Ver.4 では患者情報はレコード1、薬品情報はレコード201に格納される
	if _, ok := p.RecordMap["1"]; !ok {
		t.Error("expected record type 1 (patient info in Ver.4)")
	}
	if _, ok := p.RecordMap["201"]; !ok {
		t.Error("expected record type 201 (drug info in Ver.4)")
	}
}

func TestParse_SplitQR_Combined(t *testing.T) {
	r1 := readTestdata("ver4_split_1.txt")
	r2 := readTestdata("ver4_split_2.txt")
	p, _ := parser.Parse([]string{r1, r2})
	// Ver.4 分割QR結合後、薬品情報はレコード201に格納される
	if _, ok := p.RecordMap["201"]; !ok {
		t.Error("expected record type 201 (drug info) after combining split QRs")
	}
}

func TestParse_SplitQR_Missing(t *testing.T) {
	// QR#2 のみ提供 → QR#1 欠落の警告が出ることを確認
	r2 := readTestdata("ver4_split_only2.txt")
	_, msgs := parser.Parse([]string{r2})
	found := false
	for _, w := range msgs {
		if strings.Contains(w, "分割") {
			found = true
		}
	}
	if !found {
		t.Error("expected warning about missing split QR part")
	}
}

func TestParse_Ver3_DetectsVersion(t *testing.T) {
	raw := readTestdata("ver3_single.txt")
	p, _ := parser.Parse([]string{raw})
	if p.Version != parser.Version3 {
		t.Errorf("expected Version3, got %v", p.Version)
	}
}

func TestParse_Ver2_DetectsVersion(t *testing.T) {
	raw := readTestdata("ver2_single.txt")
	p, _ := parser.Parse([]string{raw})
	if p.Version != parser.Version2 {
		t.Errorf("expected Version2, got %v", p.Version)
	}
}

func TestParse_VersionUnknown_FallsBackToVer4(t *testing.T) {
	raw := "99,unknown\n2,テスト,テスト,19900101,1,,,"
	p, msgs := parser.Parse([]string{raw})
	if p.Version != parser.Version4 {
		t.Errorf("expected fallback to Version4, got %v", p.Version)
	}
	found := false
	for _, msg := range msgs {
		if strings.Contains(msg, "Ver.4") {
			found = true
		}
	}
	if !found {
		t.Error("expected INFO about version fallback")
	}
}
