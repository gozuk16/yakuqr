package output_test

import (
	"os"
	"strings"
	"testing"

	"github.com/gozuk16/yakuqr/pkg/output"
	"github.com/gozuk16/yakuqr/pkg/parser"
	"github.com/gozuk16/yakuqr/pkg/validator"
)

func makeTestPrescription() (parser.Prescription, []validator.ValidationResult) {
	records := []parser.Record{
		{Type: "1", Fields: []string{"1", "4", "131012345"}},
		{Type: "2", Fields: []string{"2", "山田太郎", "ヤマダタロウ", "19700101", "1"}},
		{Type: "6", Fields: []string{"6", "110626050", "アムロジピン錠5mg", "1", "錠", "", "28", "3"}},
	}
	rm := map[string][]parser.Record{
		"1": {records[0]},
		"2": {records[1]},
		"6": {records[2]},
	}
	p := parser.Prescription{
		Version:   parser.Version2_1,
		RawQRs:    []string{"1,4,131012345\n2,山田太郎,ヤマダタロウ,19700101,1"},
		Records:   records,
		RecordMap: rm,
	}
	results := []validator.ValidationResult{
		{Level: validator.LevelWarning, Field: "薬品コード", Message: "HOTコードが空です"},
	}
	return p, results
}

func TestBuildText_ContainsRawSection(t *testing.T) {
	p, results := makeTestPrescription()
	text := output.BuildText(p, results)
	if !strings.Contains(text, "=== RAW QR DATA ===") {
		t.Error("expected RAW QR DATA section")
	}
}

func TestBuildText_ContainsParsedSection(t *testing.T) {
	p, results := makeTestPrescription()
	text := output.BuildText(p, results)
	if !strings.Contains(text, "=== PARSED DATA ===") {
		t.Error("expected PARSED DATA section")
	}
}

func TestBuildText_ContainsValidationSection(t *testing.T) {
	p, results := makeTestPrescription()
	text := output.BuildText(p, results)
	if !strings.Contains(text, "=== VALIDATION RESULTS ===") {
		t.Error("expected VALIDATION RESULTS section")
	}
	if !strings.Contains(text, "[WARNING]") {
		t.Error("expected WARNING in validation section")
	}
}

func TestWriteOutput_CreatesFile(t *testing.T) {
	p, results := makeTestPrescription()
	tmpFile := t.TempDir() + "/out.txt"
	if err := output.WriteOutput(tmpFile, p, results); err != nil {
		t.Fatalf("WriteOutput: %v", err)
	}
	b, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(b), "RAW QR DATA") {
		t.Error("output file missing RAW QR DATA")
	}
}

func TestOutputPath_DefaultSuffix(t *testing.T) {
	existing := make(map[string]bool)
	got := output.OutputPath("scan.png", "", existing)
	if got != "scan_out.txt" {
		t.Errorf("expected scan_out.txt, got %s", got)
	}
}

func TestOutputPath_WithOutputDir(t *testing.T) {
	existing := make(map[string]bool)
	got := output.OutputPath("scan.png", "/out", existing)
	if got != "/out/scan_out.txt" {
		t.Errorf("expected /out/scan_out.txt, got %s", got)
	}
}

func TestOutputPath_CollisionNumbered(t *testing.T) {
	existing := map[string]bool{"scan_out.txt": true}
	got := output.OutputPath("scan.png", "", existing)
	if got != "scan_out_2.txt" {
		t.Errorf("expected scan_out_2.txt, got %s", got)
	}
}
