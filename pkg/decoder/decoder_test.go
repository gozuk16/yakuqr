package decoder_test

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestDecodeImage_SingleQR(t *testing.T) {
	path := testdataPath("qr_ver4_single.png")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("testdata not yet available")
	}
	results, errs := decoder.DecodeFile(path)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one QR result")
	}
}
