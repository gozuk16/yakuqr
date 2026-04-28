package decoder_test

import (
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
