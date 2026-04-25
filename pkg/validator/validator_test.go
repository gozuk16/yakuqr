package validator_test

import (
	"testing"

	"github.com/gozuk16/yakuqr/pkg/parser"
	"github.com/gozuk16/yakuqr/pkg/validator"
)

func makeMinimalPrescription() parser.Prescription {
	records := []parser.Record{
		{Type: "1", Fields: []string{"1", "4", "131012345"}},
		{Type: "2", Fields: []string{"2", "山田太郎", "ヤマダタロウ", "19700101", "1"}},
		{Type: "6", Fields: []string{"6", "110626050", "アムロジピン錠5mg", "1", "錠"}},
	}
	rm := map[string][]parser.Record{
		"1": {records[0]},
		"2": {records[1]},
		"6": {records[2]},
	}
	return parser.Prescription{Version: parser.Version4, Records: records, RecordMap: rm}
}

func TestValidate_ValidPrescription_NoErrors(t *testing.T) {
	p := makeMinimalPrescription()
	results := validator.Validate(p)
	for _, r := range results {
		if r.Level == validator.LevelError {
			t.Errorf("unexpected ERROR: %v", r)
		}
	}
}

func TestValidate_MissingRecord1_ReturnsError(t *testing.T) {
	p := makeMinimalPrescription()
	delete(p.RecordMap, "1")
	results := validator.Validate(p)
	found := false
	for _, r := range results {
		if r.Level == validator.LevelError && r.Field == "処方箋情報(レコード1)" {
			found = true
		}
	}
	if !found {
		t.Error("expected ERROR for missing record 1")
	}
}

func TestValidate_MissingPatientName_ReturnsError(t *testing.T) {
	p := makeMinimalPrescription()
	rec := p.RecordMap["2"][0]
	rec.Fields[1] = ""
	p.RecordMap["2"][0] = rec
	results := validator.Validate(p)
	found := false
	for _, r := range results {
		if r.Level == validator.LevelError && r.Field == "患者氏名" {
			found = true
		}
	}
	if !found {
		t.Error("expected ERROR for empty patient name")
	}
}

func TestValidate_Ver2_ReturnsInfo(t *testing.T) {
	p := makeMinimalPrescription()
	p.Version = parser.Version2
	results := validator.Validate(p)
	found := false
	for _, r := range results {
		if r.Level == validator.LevelInfo {
			found = true
		}
	}
	if !found {
		t.Error("expected INFO for Ver.2")
	}
}
