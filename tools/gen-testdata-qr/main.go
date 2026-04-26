package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	srcDir := filepath.Join(cwd, "testdata")
	dstDir := filepath.Join(cwd, "testdata", "generated")

	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	entries, err := filepath.Glob(filepath.Join(srcDir, "*.txt"))
	if err != nil || len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "no .txt files found in testdata/")
		os.Exit(1)
	}

	for _, entry := range entries {
		content, err := os.ReadFile(entry)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read %s: %v\n", entry, err)
			continue
		}

		// UTF-8 → Shift_JIS: 実際の JAHIS QR コードと同じエンコーディングで生成する
		enc := japanese.ShiftJIS.NewEncoder()
		sjisBytes, _, err := transform.Bytes(enc, content)
		if err != nil {
			fmt.Fprintf(os.Stderr, "encode %s: %v\n", entry, err)
			continue
		}

		stem := strings.TrimSuffix(filepath.Base(entry), ".txt")
		outPath := filepath.Join(dstDir, "qr_"+stem+".png")

		if err := qrcode.WriteFile(string(sjisBytes), qrcode.Medium, 256, outPath); err != nil {
			fmt.Fprintf(os.Stderr, "write QR %s: %v\n", outPath, err)
			continue
		}
		fmt.Printf("generated: %s\n", outPath)
	}
}
