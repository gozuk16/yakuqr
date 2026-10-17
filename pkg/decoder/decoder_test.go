package decoder_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gozuk16/yakuqr/pkg/decoder"
)

func testdataPath(name string) string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "testdata")
	return filepath.Join(root, name)
}

func TestDecodeFile_UnknownExtension(t *testing.T) {
	_, errs := decoder.DecodeFile("testfile.xyz")
	if len(errs) == 0 {
		t.Fatal("expected error for unknown extension")
	}
}

func TestDecodeFile_NotExist(t *testing.T) {
	_, errs := decoder.DecodeFile("notexist.png")
	if len(errs) == 0 {
		t.Fatal("expected error for non-existent file")
	}
}

var decoderImageCases = []struct {
	file       string
	wantSubstr string
}{
	{"qr_ver4_single.png", "JAHISTC04"},
	{"qr_ver4_split_1.png", "JAHISTC04,1"},
	{"qr_ver4_split_2.png", "JAHISTC04,2"},
	{"qr_ver2_single.png", "1,2,"},
	{"qr_ver3_single.png", "1,3,"},
}

func TestDecodeImage_Generated(t *testing.T) {
	for _, tc := range decoderImageCases {
		t.Run(tc.file, func(t *testing.T) {
			path := testdataPath(filepath.Join("generated", tc.file))
			results, errs := decoder.DecodeFile(path)
			if len(errs) > 0 {
				t.Fatalf("decode errors: %v", errs)
			}
			if len(results) == 0 {
				t.Fatal("no QR decoded")
			}
			if results[0].Err != nil {
				t.Fatalf("first QR decode error: %v", results[0].Err)
			}
			if !strings.Contains(results[0].Text, tc.wantSubstr) {
				t.Errorf("expected %q in decoded result, got: %q", tc.wantSubstr, results[0].Text)
			}
		})
	}
}

func TestDecodeImage_Collected(t *testing.T) {
	dir := testdataPath("collected")
	var files []string
	for _, pat := range []string{"*.jpg", "*.jpeg", "*.png"} {
		matches, _ := filepath.Glob(filepath.Join(dir, pat))
		files = append(files, matches...)
	}
	if len(files) == 0 {
		t.Skip("no collected testdata; see testdata/collected/README.md for download instructions")
	}
	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			results, errs := decoder.DecodeFile(f)
			if len(errs) > 0 {
				t.Logf("decode errors (non-fatal): %v", errs)
			}
			if len(results) == 0 {
				t.Fatal("no QR decoded from collected sample")
			}
		})
	}
}

func TestDecodeImage_StructuredAppend(t *testing.T) {
	want, err := os.ReadFile(testdataPath("ver4_single.txt"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		file         string
		wantParityNG bool
		wantReadFail bool
	}{
		{"sa_a_valid_char_straddle.png", false, false},
		{"sa_b_truncated_mid_char.png", true, false},
		{"sa_c_truncated_on_boundary.png", true, false},
		{"sa_d_overflow_declared_len.png", false, true},
		{"sa_e_misconfigured_ecl.png", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			results, errs := decoder.DecodeFile(testdataPath(filepath.Join("generated", "sa", tc.file)))
			if len(errs) > 0 {
				t.Fatalf("decode errors: %v", errs)
			}
			if len(results) != 2 {
				t.Fatalf("got %d results, want 2", len(results))
			}
			var sb strings.Builder
			readFail, parityNG := false, false
			for _, r := range results {
				if r.Err != nil {
					readFail = true
				}
				if r.SAParityErr != nil {
					parityNG = true
				}
				sb.WriteString(r.Text)
			}
			if readFail != tc.wantReadFail {
				t.Errorf("readFail = %v, want %v", readFail, tc.wantReadFail)
			}
			if parityNG != tc.wantParityNG {
				t.Errorf("parity mismatch = %v, want %v", parityNG, tc.wantParityNG)
			}
			if !tc.wantParityNG && !tc.wantReadFail {
				if got := sb.String(); got != string(want) {
					t.Errorf("concatenated text mismatch\n got: %q\nwant: %q", got, want)
				}
				for i, r := range results {
					if strings.ContainsRune(r.Text, '�') {
						t.Errorf("QR #%d contains U+FFFD: %q", i+1, r.Text)
					}
				}
			}
		})
	}
}
