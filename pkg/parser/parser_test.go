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
	if _, ok := p.RecordMap["2"]; !ok {
		t.Error("expected record type 2 (patient info)")
	}
}

func TestParse_SplitQR_Combined(t *testing.T) {
	r1 := readTestdata("ver4_split_1.txt")
	r2 := readTestdata("ver4_split_2.txt")
	p, _ := parser.Parse([]string{r1, r2})
	if _, ok := p.RecordMap["6"]; !ok {
		t.Error("expected record type 6 (drug info) after combining split QRs")
	}
}

func TestParse_SplitQR_Missing(t *testing.T) {
	r1 := readTestdata("ver4_split_1.txt")
	_, msgs := parser.Parse([]string{r1})
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
