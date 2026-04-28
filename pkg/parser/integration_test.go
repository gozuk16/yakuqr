package parser_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gozuk16/yakuqr/pkg/decoder"
	"github.com/gozuk16/yakuqr/pkg/parser"
	"github.com/gozuk16/yakuqr/pkg/validator"
)

func toRawQRs(results []decoder.QRResult) []parser.RawQR {
	raws := make([]parser.RawQR, len(results))
	for i, r := range results {
		if r.Err != nil {
			raws[i] = parser.RawQR{ErrMsg: r.Err.Error()}
		} else {
			raws[i] = parser.RawQR{Text: r.Text}
		}
	}
	return raws
}

func generatedImagePath(name string) string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "generated")
	return filepath.Join(root, name)
}

var pipelineCases = []struct {
	imageFile   string
	wantVersion parser.Version
	wantRecord  string
}{
	{"qr_ver4_single.png", parser.Version2_1, "1"},
	{"qr_ver2_single.png", parser.Version1_1, "2"},
	{"qr_ver3_single.png", parser.Version2_0, "2"},
}

func TestPipeline_DecodeParseValidate(t *testing.T) {
	for _, tc := range pipelineCases {
		t.Run(tc.imageFile, func(t *testing.T) {
			path := generatedImagePath(tc.imageFile)

			qrResults, errs := decoder.DecodeFile(path)
			if len(errs) > 0 {
				t.Fatalf("decode errors: %v", errs)
			}
			if len(qrResults) == 0 {
				t.Fatal("no QR decoded")
			}

			rawQRs := toRawQRs(qrResults)
			p, _ := parser.Parse(rawQRs)
			if p.Version != tc.wantVersion {
				t.Errorf("version: want %v, got %v", tc.wantVersion, p.Version)
			}
			if _, ok := p.RecordMap[tc.wantRecord]; !ok {
				t.Errorf("RecordMap[%q] not found", tc.wantRecord)
			}

			results := validator.Validate(p)
			for _, r := range results {
				if r.Level == validator.LevelError {
					t.Errorf("unexpected validation ERROR: %v", r)
				}
			}
		})
	}
}

func TestPipeline_SplitQR_Combined(t *testing.T) {
	path1 := generatedImagePath("qr_ver4_split_1.png")
	path2 := generatedImagePath("qr_ver4_split_2.png")

	qrs1, errs1 := decoder.DecodeFile(path1)
	if len(errs1) > 0 {
		t.Fatalf("decode split_1 errors: %v", errs1)
	}
	qrs2, errs2 := decoder.DecodeFile(path2)
	if len(errs2) > 0 {
		t.Fatalf("decode split_2 errors: %v", errs2)
	}

	allQRs := toRawQRs(append(qrs1, qrs2...))

	p, msgs := parser.Parse(allQRs)

	for _, m := range msgs {
		t.Logf("parse message: %s", m)
	}

	if p.Version != parser.Version2_1 {
		t.Errorf("version: want Version2_1, got %v", p.Version)
	}
	if _, ok := p.RecordMap["201"]; !ok {
		t.Error("RecordMap[\"201\"] not found — split QR combination may have failed")
	}

	results := validator.Validate(p)
	for _, r := range results {
		if r.Level == validator.LevelError {
			t.Errorf("unexpected validation ERROR: %v", r)
		}
	}
}

func TestPipeline_911Split_Combined(t *testing.T) {
	path1 := generatedImagePath("qr_ver2_6_911split_1.png")
	path2 := generatedImagePath("qr_ver2_6_911split_2.png")

	qrs1, errs1 := decoder.DecodeFile(path1)
	if len(errs1) > 0 {
		t.Fatalf("decode 911split_1 errors: %v", errs1)
	}
	qrs2, errs2 := decoder.DecodeFile(path2)
	if len(errs2) > 0 {
		t.Fatalf("decode 911split_2 errors: %v", errs2)
	}

	allQRs := toRawQRs(append(qrs1, qrs2...))
	p, msgs := parser.Parse(allQRs)
	for _, m := range msgs {
		t.Logf("parse message: %s", m)
	}

	if p.Version != parser.Version2_6 {
		t.Errorf("version: want Version2_6, got %v", p.Version)
	}

	// 911分割: QR2（累積）から両方のRpが取得できること
	rp201 := p.RecordMap["201"]
	if len(rp201) < 2 {
		t.Errorf("201レコード数: want >= 2 (Rp1+Rp2), got %d", len(rp201))
	}

	// 911レコード自体はRecordMapに残らないこと
	if _, ok := p.RecordMap["911"]; ok {
		t.Error("911レコードがRecordMapに残っている（remove911Linesが機能していない）")
	}

	// SplitInfosに911分割情報が格納されていること
	if len(p.SplitInfos) != 2 {
		t.Errorf("SplitInfos len: want 2, got %d", len(p.SplitInfos))
	}
	if p.SplitInfos[0].Total != 2 || p.SplitInfos[1].Total != 2 {
		t.Errorf("SplitInfos.Total: want 2, got %v", p.SplitInfos)
	}

	results := validator.Validate(p)
	for _, r := range results {
		if r.Level == validator.LevelError {
			t.Errorf("unexpected validation ERROR: %v", r)
		}
	}
}
