package validator_test

import (
	"strings"
	"testing"

	"github.com/gozuk16/yakuqr/pkg/parser"
	"github.com/gozuk16/yakuqr/pkg/validator"
)

// makeMinimalPrescriptionVer4 は Ver.4 形式の最小限の処方箋を生成する。
// Ver.4: レコード1=患者情報(fields[1]=氏名, fields[3]=生年月日)、レコード201=薬品情報
func makeMinimalPrescriptionVer4() parser.Prescription {
	records := []parser.Record{
		{Type: "1", Fields: []string{"1", "山田太郎", "1", "19700101", "100-0001"}},
		{Type: "201", Fields: []string{"201", "1", "アムロジピン錠5mg", "1", "錠"}},
	}
	rm := map[string][]parser.Record{
		"1":   {records[0]},
		"201": {records[1]},
	}
	return parser.Prescription{Version: parser.Version2_1, Records: records, RecordMap: rm}
}

// makeMinimalPrescriptionVer2 は Ver.2 形式の最小限の処方箋を生成する。
// Ver.2: レコード1=処方箋情報、レコード2=患者情報(fields[1]=氏名, fields[3]=生年月日)、レコード6=薬品情報
func makeMinimalPrescriptionVer2() parser.Prescription {
	records := []parser.Record{
		{Type: "1", Fields: []string{"1", "2", "131012345"}},
		{Type: "2", Fields: []string{"2", "山田太郎", "ヤマダタロウ", "19700101", "1"}},
		{Type: "6", Fields: []string{"6", "110626050", "アムロジピン錠5mg", "1", "錠"}},
	}
	rm := map[string][]parser.Record{
		"1": {records[0]},
		"2": {records[1]},
		"6": {records[2]},
	}
	return parser.Prescription{Version: parser.Version1_1, Records: records, RecordMap: rm}
}

func TestValidate_ValidPrescription_NoErrors(t *testing.T) {
	p := makeMinimalPrescriptionVer4()
	results := validator.Validate(p)
	for _, r := range results {
		if r.Level == validator.LevelError {
			t.Errorf("unexpected ERROR: %v", r)
		}
	}
}

func TestValidate_MissingRecord1_ReturnsError(t *testing.T) {
	p := makeMinimalPrescriptionVer4()
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
	p := makeMinimalPrescriptionVer4()
	// Ver.4 では患者氏名はレコード1のフィールド2
	rec := p.RecordMap["1"][0]
	rec.Fields[1] = ""
	p.RecordMap["1"][0] = rec
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
	p := makeMinimalPrescriptionVer2()
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

func TestValidate_UnknownVersion_ReturnsError(t *testing.T) {
	p := parser.Prescription{
		Version:   parser.VersionUnknown,
		Records:   []parser.Record{},
		RecordMap: map[string][]parser.Record{},
	}
	results := validator.Validate(p)
	found := false
	for _, r := range results {
		if r.Level == validator.LevelError && strings.Contains(r.Message, "バージョン") {
			found = true
		}
	}
	if !found {
		t.Error("expected ERROR for unknown version")
	}
}
