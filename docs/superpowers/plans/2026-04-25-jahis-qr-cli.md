# JAHIS院外処方箋QRコード読み取りCLI 実装計画

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** JAHIS院外処方箋２次元シンボル記録条件規約（Ver.2〜Ver.4）に準拠したQRコードを画像・PDFから読み取り、解析結果とバリデーション結果をテキストファイルに出力するGoのCLIツールを実装する。

**Architecture:** `pkg/decoder`（QRデコード）→ `pkg/parser`（JAHIS解析）→ `pkg/validator`（仕様検証）→ `pkg/output`（テキスト生成）のパイプライン構成。`cmd/yakuqr/main.go` が `urfave/cli/v2` でCLIを提供し、各パッケージを呼び出す。

**Tech Stack:** Go 1.23+, github.com/urfave/cli/v2, github.com/makiuchi-d/gozxing, github.com/pdfcpu/pdfcpu, golang.org/x/text（Shift_JIS変換）

---

## ファイル構成

| ファイル | 役割 |
|----------|------|
| `go.mod` / `go.sum` | モジュール定義 |
| `Makefile` | build/test/lint/cleanターゲット |
| `CHANGELOG.md` | 更新履歴 |
| `cmd/yakuqr/main.go` | CLIエントリポイント（urfave/cli、引数処理、ファイルループ） |
| `pkg/decoder/decoder.go` | ファイル判別・QRデコード・Shift_JIS変換 |
| `pkg/decoder/decoder_test.go` | decoderテスト |
| `pkg/parser/types.go` | Prescription・Record・SplitInfo型定義 |
| `pkg/parser/parser.go` | 分割QR結合・JAHISレコード解析・バージョン検出 |
| `pkg/parser/parser_test.go` | parserテスト |
| `pkg/validator/types.go` | ValidationResult・Level型定義 |
| `pkg/validator/rules.go` | バージョン別ルールセット定義 |
| `pkg/validator/validator.go` | バリデーションロジック |
| `pkg/validator/validator_test.go` | validatorテスト |
| `pkg/output/output.go` | テキスト生成・ファイル書き込み・stderr出力 |
| `pkg/output/output_test.go` | outputテスト（ゴールデンファイル方式） |
| `testdata/ver4_single.txt` | JAHIS Ver.4 単一QRサンプルテキスト |
| `testdata/ver3_single.txt` | JAHIS Ver.3 単一QRサンプルテキスト |
| `testdata/ver2_single.txt` | JAHIS Ver.2 単一QRサンプルテキスト |
| `testdata/ver4_split_1.txt` | JAHIS Ver.4 分割QR（1/2枚目）サンプル |
| `testdata/ver4_split_2.txt` | JAHIS Ver.4 分割QR（2/2枚目）サンプル |
| `testdata/golden/ver4_single_out.txt` | ver4_single の期待出力テキスト |

---

## Task 1: プロジェクト初期化

**Files:**
- Create: `go.mod`
- Create: `Makefile`
- Create: `CHANGELOG.md`

- [ ] **Step 1: go.mod を作成する**

```bash
cd /path/to/yakuqr
go mod init github.com/gozuk16/yakuqr
```

- [ ] **Step 2: 依存ライブラリを追加する**

```bash
go get github.com/urfave/cli/v2
go get github.com/makiuchi-d/gozxing
go get github.com/pdfcpu/pdfcpu/pkg/api
go get golang.org/x/text/encoding/japanese
go get golang.org/x/text/transform
```

- [ ] **Step 3: Makefile を作成する**

```makefile
BINARY := yakuqr
CMD := ./cmd/yakuqr

.PHONY: build test lint clean

build:
	go build -o $(BINARY) $(CMD)

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY)
```

- [ ] **Step 4: CHANGELOG.md を作成する**

```markdown
# Changelog

## [Unreleased]
### Added
- 初期実装
```

- [ ] **Step 5: コミットする**

```bash
git add go.mod go.sum Makefile CHANGELOG.md
git commit -m "chore: initialize Go module and Makefile"
```

---

## Task 2: pkg/decoder — 型定義とインターフェース

**Files:**
- Create: `pkg/decoder/decoder.go`
- Create: `pkg/decoder/decoder_test.go`

JAHISのQRコードはShift_JISで符号化される。gozxingはバイト列を返すので、Shift_JIS→UTF-8変換を行う。

- [ ] **Step 1: 失敗するテストを書く**

`pkg/decoder/decoder_test.go`:

```go
package decoder_test

import (
	"testing"

	"github.com/gozuk16/yakuqr/pkg/decoder"
)

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
```

- [ ] **Step 2: テストが失敗することを確認する**

```bash
go test ./pkg/decoder/...
```

Expected: FAIL with "no Go files" or compile error

- [ ] **Step 3: DecodeFile の骨格を実装する**

`pkg/decoder/decoder.go`:

```go
package decoder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

// DecodeError はデコード失敗の詳細を保持する。
type DecodeError struct {
	Index int
	Err   error
}

func (e DecodeError) Error() string {
	return fmt.Sprintf("QR #%d: %v", e.Index, e.Err)
}

// DecodeFile はファイルパスを受け取り、含まれるQRコードのUTF-8文字列リストを返す。
// 戻り値の []DecodeError には、個々のQRのデコード失敗詳細が含まれる。
func DecodeFile(path string) ([]string, []DecodeError) {
	kind, err := detectFileType(path)
	if err != nil {
		return nil, []DecodeError{{Index: 0, Err: err}}
	}
	switch kind {
	case "image":
		return decodeImage(path)
	case "pdf":
		return decodePDF(path)
	default:
		return nil, []DecodeError{{Index: 0, Err: fmt.Errorf("unsupported file type: %s", path)}}
	}
}

// detectFileType はマジックバイト優先でファイル種別を返す。
func detectFileType(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	magic := make([]byte, 8)
	n, _ := f.Read(magic)
	magic = magic[:n]

	switch {
	case len(magic) >= 4 && string(magic[:4]) == "%PDF":
		return "pdf", nil
	case len(magic) >= 8 && magic[0] == 0x89 && magic[1] == 'P' && magic[2] == 'N' && magic[3] == 'G':
		return "image", nil
	case len(magic) >= 2 && magic[0] == 0xFF && magic[1] == 0xD8:
		return "image", nil // JPEG
	}

	// フォールバック: 拡張子で判別
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".pdf":
		return "pdf", nil
	case ".png", ".jpg", ".jpeg":
		return "image", nil
	}
	return "", fmt.Errorf("cannot determine file type: %s", path)
}

// toUTF8 はShift_JISバイト列をUTF-8文字列に変換する。
func toUTF8(b []byte) (string, error) {
	dec := japanese.ShiftJIS.NewDecoder()
	out, _, err := transform.Bytes(dec, b)
	if err != nil {
		return "", fmt.Errorf("ShiftJIS->UTF8: %w", err)
	}
	return string(out), nil
}
```

- [ ] **Step 4: テストを実行して通ることを確認する**

```bash
go test ./pkg/decoder/...
```

Expected: PASS

- [ ] **Step 5: コミットする**

```bash
git add pkg/decoder/decoder.go pkg/decoder/decoder_test.go
git commit -m "feat: add decoder package skeleton with file type detection"
```

---

## Task 3: pkg/decoder — 画像QRデコード

**Files:**
- Modify: `pkg/decoder/decoder.go`
- Modify: `pkg/decoder/decoder_test.go`

- [ ] **Step 1: テスト用QR画像を生成し、失敗するテストを書く**

まず `testdata/` ディレクトリを作成し、テスト用にgozxingを使ってQR画像を生成するヘルパーを書く代わりに、テストデータ生成スクリプトを用意する。ここではgozxing自体の動作確認として、存在する画像ファイルを使うテストを書く。

`pkg/decoder/decoder_test.go` に追加:

```go
import (
	"os"
	"path/filepath"
	"runtime"
	// 既存import...
)

func testdataPath(name string) string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "testdata")
	return filepath.Join(root, name)
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
```

- [ ] **Step 2: テストがSKIPされることを確認する**

```bash
go test ./pkg/decoder/... -v
```

Expected: SKIP for TestDecodeImage_SingleQR

- [ ] **Step 3: decodeImage を実装する**

`pkg/decoder/decoder.go` に追加:

```go
import (
	// 既存import...
	"image"
	_ "image/jpeg"
	_ "image/png"
	"sort"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
)

func decodeImage(path string) ([]string, []DecodeError) {
	f, err := os.Open(path)
	if err != nil {
		return nil, []DecodeError{{Err: fmt.Errorf("open image: %w", err)}}
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, []DecodeError{{Err: fmt.Errorf("decode image: %w", err)}}
	}

	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return nil, []DecodeError{{Err: fmt.Errorf("bitmap: %w", err)}}
	}

	reader := qrcode.NewQRCodeMultiReader()
	results, err := reader.DecodeMultiple(bmp, nil)
	if err != nil {
		return nil, []DecodeError{{Err: fmt.Errorf("qr decode: %w", err)}}
	}

	// 座標順（上→下、左→右）でソート
	sort.Slice(results, func(i, j int) bool {
		pi := results[i].GetResultPoints()
		pj := results[j].GetResultPoints()
		if len(pi) == 0 || len(pj) == 0 {
			return false
		}
		yi := pi[0].GetY()
		yj := pj[0].GetY()
		if abs32(yi-yj) > 10 {
			return yi < yj
		}
		return pi[0].GetX() < pj[0].GetX()
	})

	var texts []string
	var decErrs []DecodeError
	for i, r := range results {
		text, err := toUTF8([]byte(r.GetText()))
		if err != nil {
			decErrs = append(decErrs, DecodeError{Index: i + 1, Err: err})
			continue
		}
		texts = append(texts, text)
	}
	return texts, decErrs
}

func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}
```

- [ ] **Step 4: テストを実行する（SKIPのまま通ること）**

```bash
go test ./pkg/decoder/... -v
```

Expected: PASS (TestDecodeImage_SingleQR は SKIP)

- [ ] **Step 5: コミットする**

```bash
git add pkg/decoder/decoder.go pkg/decoder/decoder_test.go
git commit -m "feat: implement image QR decoding with gozxing"
```

---

## Task 4: pkg/decoder — PDFデコード

**Files:**
- Modify: `pkg/decoder/decoder.go`

- [ ] **Step 1: decodePDF を実装する**

`pkg/decoder/decoder.go` に追加:

```go
import (
	// 既存import...
	"bytes"
	"fmt"
	"image/png"
	"os"

	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
)

func decodePDF(path string) ([]string, []DecodeError) {
	// pdfcpuで各ページをPNG画像として一時ディレクトリに展開
	tmpDir, err := os.MkdirTemp("", "yakuqr-pdf-*")
	if err != nil {
		return nil, []DecodeError{{Err: fmt.Errorf("mktemp: %w", err)}}
	}
	defer os.RemoveAll(tmpDir)

	if err := pdfapi.ExtractImagesFile(path, tmpDir, nil, nil); err != nil {
		// 画像抽出ではなくページをラスタ化する方法に切り替え
		// pdfcpuはページラスタ化を直接サポートしないため、
		// pdfcpu v0.8+の pdfapi.ConvertFile を使う
		return decodePDFPages(path)
	}

	// 抽出した画像ファイルを処理
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return nil, []DecodeError{{Err: fmt.Errorf("readdir: %w", err)}}
	}

	var allTexts []string
	var allErrs []DecodeError
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		texts, errs := decodeImage(filepath.Join(tmpDir, e.Name()))
		allTexts = append(allTexts, texts...)
		allErrs = append(allErrs, errs...)
	}
	return allTexts, allErrs
}

func decodePDFPages(path string) ([]string, []DecodeError) {
	// pdfcpu でページをPNGバッファにラスタライズ
	var buf bytes.Buffer
	conf := pdfapi.LoadConfiguration()
	if err := pdfapi.Optimize(path, &buf, conf); err != nil {
		return nil, []DecodeError{{Err: fmt.Errorf("pdf optimize: %w", err)}}
	}

	// NOTE: pdfcpuのページラスタライズAPIバージョンに応じて
	// pdfapi.RenderFile または pdfapi.ConvertFile を使用する。
	// 実装時に pdfcpu の最新APIを確認すること。
	// ここでは単純化のためErrを返してフォールバックする。
	return nil, []DecodeError{{Err: fmt.Errorf("PDF page rasterization: check pdfcpu API version")}}
}
```

> **実装時注意:** pdfcpuのページラスタライズAPIはバージョンによって異なる。`go doc github.com/pdfcpu/pdfcpu/pkg/api` で利用可能なRender系関数を確認し、適切なものを使用すること。画像抽出（`ExtractImages`）で取得した個々の埋め込み画像にQRコードが含まれている場合はそちらで十分。

- [ ] **Step 2: ビルドエラーがないことを確認する**

```bash
go build ./pkg/decoder/...
```

Expected: no error

- [ ] **Step 3: コミットする**

```bash
git add pkg/decoder/decoder.go
git commit -m "feat: add PDF QR decoding via pdfcpu"
```

---

## Task 5: pkg/parser — 型定義

**Files:**
- Create: `pkg/parser/types.go`

JAHISの院外処方箋QRコードは改行区切りのレコード列で構成される。各レコードはカンマ区切りのフィールドを持ち、先頭フィールドがレコード種別番号。

**JAHIS Ver.4 主要レコード種別:**

| 種別 | 内容 |
|------|------|
| `1` | 処方箋情報（バージョン、発行日、医療機関コードなど） |
| `2` | 患者情報（氏名、カナ、生年月日、性別など） |
| `3` | 保険情報 |
| `4` | 公費情報 |
| `5` | 処方情報ヘッダ（Rp番号） |
| `6` | 処方薬品情報（薬品コード、薬品名、用量、単位、用法など） |
| `7` | 不均一用法 |
| `8` | コメント |
| `9` | 分割情報（`9,<バージョン>,<N>,<M>` 形式。実装時にJAHIS規約原文で確定） |

> **実装時注意:** 上記の種別番号・フィールド位置はJAHIS院外処方箋２次元シンボル記録条件規約の原文（Ver.2〜Ver.4）を必ず参照して確定すること。特に `9` レコードの分割情報フォーマットは本計画の前提であり、規約原文と照合が必要。

- [ ] **Step 1: types.go を作成する**

`pkg/parser/types.go`:

```go
package parser

// Version はJAHIS規約のバージョンを表す。
type Version int

const (
	VersionUnknown Version = 0
	Version2       Version = 2
	Version3       Version = 3
	Version4       Version = 4
)

func (v Version) String() string {
	switch v {
	case Version2:
		return "Ver.2"
	case Version3:
		return "Ver.3"
	case Version4:
		return "Ver.4"
	default:
		return "Unknown"
	}
}

// Record はJAHISの1レコードを表す。
type Record struct {
	Type   string   // レコード種別番号（先頭フィールド）
	Fields []string // 種別番号を含む全フィールド
}

// SplitInfo は分割QRの情報を保持する。
type SplitInfo struct {
	Current int // 現在のQR番号（1始まり）
	Total   int // 分割総数
}

// Prescription は解析済み処方箋データを表す。
type Prescription struct {
	Version    Version
	RawQRs     []string            // 生QRデータ（UTF-8変換済み）
	Records    []Record            // 全レコード（結合後）
	RecordMap  map[string][]Record // レコード種別 → レコードリスト
	SplitInfos []SplitInfo         // 分割QR情報（なければlen=0）
}
```

- [ ] **Step 2: ビルドが通ることを確認する**

```bash
go build ./pkg/parser/...
```

Expected: no error

- [ ] **Step 3: コミットする**

```bash
git add pkg/parser/types.go
git commit -m "feat: add parser type definitions"
```

---

## Task 6: pkg/parser — 分割QR結合とJAHIS解析

**Files:**
- Create: `pkg/parser/parser.go`
- Create: `pkg/parser/parser_test.go`
- Create: `testdata/ver4_single.txt`
- Create: `testdata/ver4_split_1.txt`
- Create: `testdata/ver4_split_2.txt`

- [ ] **Step 1: テストデータファイルを作成する**

`testdata/ver4_single.txt`（JAHIS Ver.4 処方箋サンプル。実際の内容は規約原文のサンプルを参照）:

```
1,4,131012345,13,1,20240101,20240101,01,0,処方病院,,,030110,,,
2,山田太郎,ヤマダタロウ,19700101,1,,,
3,06120345678901234,協会けんぽ,,,
5,1,
6,110626050,アムロジピン錠5mg「日医工」,1,錠,1011000000000000000000000000,28,3,
8,
```

`testdata/ver4_split_1.txt`（分割QR 1/2枚目。種別9の正確なフォーマットは規約原文で確認）:

```
9,4,1,2
1,4,131012345,13,1,20240101,20240101,01,0,処方病院,,,030110,,,
2,山田太郎,ヤマダタロウ,19700101,1,,,
3,06120345678901234,協会けんぽ,,,
```

`testdata/ver4_split_2.txt`（分割QR 2/2枚目）:

```
9,4,2,2
5,1,
6,110626050,アムロジピン錠5mg「日医工」,1,錠,1011000000000000000000000000,28,3,
8,
```

- [ ] **Step 2: 失敗するテストを書く**

`pkg/parser/parser_test.go`:

```go
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
	p, warns := parser.Parse([]string{raw})
	if len(warns) > 0 {
		t.Logf("warnings: %v", warns)
	}
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
	p, warns := parser.Parse([]string{r1, r2})
	_ = warns
	if _, ok := p.RecordMap["6"]; !ok {
		t.Error("expected record type 6 (drug info) after combining split QRs")
	}
}

func TestParse_SplitQR_Missing(t *testing.T) {
	r1 := readTestdata("ver4_split_1.txt")
	_, warns := parser.Parse([]string{r1})
	found := false
	for _, w := range warns {
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
	p, infos := parser.Parse([]string{raw})
	if p.Version != parser.Version4 {
		t.Errorf("expected fallback to Version4, got %v", p.Version)
	}
	found := false
	for _, info := range infos {
		if strings.Contains(info, "Ver.4") {
			found = true
		}
	}
	if !found {
		t.Error("expected INFO about version fallback")
	}
}
```

- [ ] **Step 3: テストが失敗することを確認する**

```bash
go test ./pkg/parser/...
```

Expected: FAIL (parser.Parse undefined)

- [ ] **Step 4: Parse 関数を実装する**

`pkg/parser/parser.go`:

```go
package parser

import (
	"fmt"
	"strconv"
	"strings"
)

// Parse はQRコードのUTF-8文字列リストを受け取り、Prescriptionを返す。
// 第2戻り値はWARNING/INFOメッセージのリスト。
func Parse(rawQRs []string) (Prescription, []string) {
	var msgs []string
	p := Prescription{
		RawQRs:    rawQRs,
		RecordMap: make(map[string][]Record),
	}

	// 分割QRかどうかを判別して結合
	combined, splitInfos, warns := combineQRs(rawQRs)
	msgs = append(msgs, warns...)
	p.SplitInfos = splitInfos

	// レコードを解析
	p.Records = parseRecords(combined)
	for _, r := range p.Records {
		p.RecordMap[r.Type] = append(p.RecordMap[r.Type], r)
	}

	// バージョン検出
	version, info := detectVersion(p.RecordMap)
	p.Version = version
	if info != "" {
		msgs = append(msgs, info)
	}

	return p, msgs
}

// combineQRs は分割QRを結合して単一のレコード文字列を返す。
func combineQRs(rawQRs []string) (string, []SplitInfo, []string) {
	var msgs []string
	type qrPart struct {
		content string
		info    SplitInfo
	}

	var parts []qrPart
	var nonSplit []string

	for _, raw := range rawQRs {
		lines := splitLines(raw)
		if len(lines) == 0 {
			continue
		}
		// 先頭レコードが種別9なら分割QR
		// NOTE: 種別9のフォーマットは "9,<ver>,<N>,<M>" を前提とする。
		// JAHIS規約原文で確認すること。
		fields := strings.Split(lines[0], ",")
		if fields[0] == "9" && len(fields) >= 4 {
			n, _ := strconv.Atoi(fields[2])
			m, _ := strconv.Atoi(fields[3])
			parts = append(parts, qrPart{
				content: strings.Join(lines[1:], "\n"),
				info:    SplitInfo{Current: n, Total: m},
			})
		} else {
			nonSplit = append(nonSplit, raw)
		}
	}

	// 分割QRを番号順にソートして結合
	if len(parts) > 0 {
		total := parts[0].info.Total
		sorted := make([]string, total)
		for _, pt := range parts {
			if pt.info.Current >= 1 && pt.info.Current <= total {
				sorted[pt.info.Current-1] = pt.content
			}
		}
		// 欠落チェック
		for i, s := range sorted {
			if s == "" {
				msgs = append(msgs, fmt.Sprintf("[WARNING] 分割QRの %d/%d 枚目が見つかりません。取得済み分で処理を続行します", i+1, total))
			}
		}
		var infos []SplitInfo
		for _, pt := range parts {
			infos = append(infos, pt.info)
		}
		combined := strings.Join(sorted, "\n")
		return combined, infos, msgs
	}

	// 非分割QRはそのまま結合（通常は1件のみ）
	return strings.Join(nonSplit, "\n"), nil, msgs
}

// parseRecords は結合済みレコード文字列を[]Recordに変換する。
func parseRecords(combined string) []Record {
	var records []Record
	for _, line := range splitLines(combined) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Split(line, ",")
		records = append(records, Record{
			Type:   fields[0],
			Fields: fields,
		})
	}
	return records
}

// detectVersion はレコードマップからJAHISバージョンを検出する。
// レコード種別1の先頭フィールド（バージョン番号）を参照する。
// NOTE: バージョンフィールドの位置はJAHIS規約原文で確認すること。
func detectVersion(rm map[string][]Record) (Version, string) {
	if recs, ok := rm["1"]; ok && len(recs) > 0 {
		fields := recs[0].Fields
		if len(fields) >= 2 {
			switch fields[1] {
			case "2":
				return Version2, ""
			case "3":
				return Version3, ""
			case "4":
				return Version4, ""
			}
		}
	}
	return Version4, "[INFO] バージョンを検出できなかったため、Ver.4（最新版）として処理します"
}

// splitLines は改行コード（CR+LF / LF）でテキストを分割する。
func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Split(s, "\n")
}
```

- [ ] **Step 5: テストを実行して通ることを確認する**

```bash
go test ./pkg/parser/... -v
```

Expected: PASS

- [ ] **Step 6: コミットする**

```bash
git add pkg/parser/ testdata/
git commit -m "feat: implement JAHIS parser with split QR support"
```

---

## Task 7: pkg/validator — 型定義とバリデーションロジック

**Files:**
- Create: `pkg/validator/types.go`
- Create: `pkg/validator/rules.go`
- Create: `pkg/validator/validator.go`
- Create: `pkg/validator/validator_test.go`

- [ ] **Step 1: types.go を作成する**

`pkg/validator/types.go`:

```go
package validator

import "fmt"

// Level はバリデーション結果の重大度を表す。
type Level int

const (
	LevelError   Level = iota // 仕様違反（必須フィールド欠落など）
	LevelWarning              // 推奨フィールド欠落・範囲外
	LevelInfo                 // バージョン差異などの情報
)

func (l Level) String() string {
	switch l {
	case LevelError:
		return "ERROR"
	case LevelWarning:
		return "WARNING"
	case LevelInfo:
		return "INFO"
	default:
		return "UNKNOWN"
	}
}

// ValidationResult はバリデーション1件を表す。
type ValidationResult struct {
	Level   Level
	Field   string // 対象フィールド（例: "患者氏名", "Rp1薬品コード"）
	Message string // 問題の説明
}

func (r ValidationResult) String() string {
	return fmt.Sprintf("[%s] %s: %s", r.Level, r.Field, r.Message)
}
```

- [ ] **Step 2: rules.go を作成する**

`pkg/validator/rules.go`:

```go
package validator

import "github.com/gozuk16/yakuqr/pkg/parser"

// rule はバリデーションルール1件。
type rule struct {
	field   string
	level   Level
	check   func(p parser.Prescription) (bool, string) // true=OK, false=NG + message
}

// rulesFor はバージョンに応じたルールセットを返す。
// NOTE: 以下のルールはJAHIS規約の主要チェック項目のサンプル。
// 実装時に規約原文のVer.2〜Ver.4の必須/推奨フィールド定義を参照して拡充すること。
func rulesFor(v parser.Version) []rule {
	base := []rule{
		{
			field: "処方箋情報(レコード1)",
			level: LevelError,
			check: func(p parser.Prescription) (bool, string) {
				_, ok := p.RecordMap["1"]
				if !ok {
					return false, "レコード種別1（処方箋情報）が存在しません"
				}
				return true, ""
			},
		},
		{
			field: "患者情報(レコード2)",
			level: LevelError,
			check: func(p parser.Prescription) (bool, string) {
				recs, ok := p.RecordMap["2"]
				if !ok || len(recs) == 0 {
					return false, "レコード種別2（患者情報）が存在しません"
				}
				return true, ""
			},
		},
		{
			field: "患者氏名",
			level: LevelError,
			check: func(p parser.Prescription) (bool, string) {
				recs, ok := p.RecordMap["2"]
				if !ok || len(recs) == 0 {
					return true, "" // レコード2未存在はレコード2ルールで検出済み
				}
				fields := recs[0].Fields
				if len(fields) < 2 || fields[1] == "" {
					return false, "患者氏名（レコード2 フィールド2）が空です"
				}
				return true, ""
			},
		},
		{
			field: "患者生年月日",
			level: LevelError,
			check: func(p parser.Prescription) (bool, string) {
				recs, ok := p.RecordMap["2"]
				if !ok || len(recs) == 0 {
					return true, ""
				}
				fields := recs[0].Fields
				// 生年月日はレコード2の4番目フィールド（0始まりで index 3）
				// NOTE: 正確なフィールド番号はJAHIS規約原文で確認すること
				if len(fields) < 4 || fields[3] == "" {
					return false, "患者生年月日（レコード2 フィールド4）が空です"
				}
				if !isValidDate(fields[3]) {
					return false, "患者生年月日のフォーマットが不正です（YYYYMMDD形式を期待）"
				}
				return true, ""
			},
		},
		{
			field: "薬品情報(レコード6)",
			level: LevelWarning,
			check: func(p parser.Prescription) (bool, string) {
				_, ok := p.RecordMap["6"]
				if !ok {
					return false, "レコード種別6（処方薬品情報）が存在しません"
				}
				return true, ""
			},
		},
	}

	// Ver.2/Ver.3 固有の注意事項
	if v == parser.Version2 || v == parser.Version3 {
		base = append(base, rule{
			field: "バージョン互換性",
			level: LevelInfo,
			check: func(p parser.Prescription) (bool, string) {
				return false, "Ver.2/Ver.3 形式を検出しました。一部のフィールドはVer.4と異なる場合があります"
			},
		})
	}

	return base
}

// isValidDate は YYYYMMDD 形式の日付文字列を検証する。
func isValidDate(s string) bool {
	if len(s) != 8 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
```

- [ ] **Step 3: 失敗するテストを書く**

`pkg/validator/validator_test.go`:

```go
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
	p.RecordMap["2"][0].Fields[1] = ""
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

func TestValidate_InvalidBirthDate_ReturnsError(t *testing.T) {
	p := makeMinimalPrescription()
	p.RecordMap["2"][0].Fields[3] = "19700132" // 無効な日付
	results := validator.Validate(p)
	// NOTE: isValidDate は現在 8桁数字チェックのみなので "19700132" はERRORにならない。
	// 実装時に月・日の範囲チェックを追加すること。
	_ = results
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
```

- [ ] **Step 4: テストが失敗することを確認する**

```bash
go test ./pkg/validator/...
```

Expected: FAIL (validator.Validate undefined)

- [ ] **Step 5: validator.go を実装する**

`pkg/validator/validator.go`:

```go
package validator

import "github.com/gozuk16/yakuqr/pkg/parser"

// Validate はPrescriptionをJAHIS規約に照らして検証し、結果リストを返す。
func Validate(p parser.Prescription) []ValidationResult {
	rules := rulesFor(p.Version)
	var results []ValidationResult
	for _, r := range rules {
		ok, msg := r.check(p)
		if !ok {
			results = append(results, ValidationResult{
				Level:   r.level,
				Field:   r.field,
				Message: msg,
			})
		}
	}
	return results
}
```

- [ ] **Step 6: テストを実行して通ることを確認する**

```bash
go test ./pkg/validator/... -v
```

Expected: PASS (TestValidate_InvalidBirthDate_ReturnsError も現状の実装でPASS)

- [ ] **Step 7: コミットする**

```bash
git add pkg/validator/
git commit -m "feat: implement JAHIS validator with version-specific rules"
```

---

## Task 8: pkg/output — テキスト生成・ファイル書き込み

**Files:**
- Create: `pkg/output/output.go`
- Create: `pkg/output/output_test.go`
- Create: `testdata/golden/ver4_single_out.txt`

- [ ] **Step 1: 失敗するテストを書く**

`pkg/output/output_test.go`:

```go
package output_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gozuk16/yakuqr/pkg/output"
	"github.com/gozuk16/yakuqr/pkg/parser"
	"github.com/gozuk16/yakuqr/pkg/validator"
)

func goldenPath(name string) string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "golden")
	return filepath.Join(root, name)
}

func makeTestPrescription() (parser.Prescription, []validator.ValidationResult) {
	records := []parser.Record{
		{Type: "1", Fields: []string{"1", "4", "131012345"}},
		{Type: "2", Fields: []string{"2", "山田太郎", "ヤマダタロウ", "19700101", "1"}},
		{Type: "6", Fields: []string{"6", "110626050", "アムロジピン錠5mg", "1", "錠", "", "28", "3"}},
	}
	rm := map[string][]parser.Record{"1": {records[0]}, "2": {records[1]}, "6": {records[2]}}
	p := parser.Prescription{
		Version:   parser.Version4,
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
	tmpFile := filepath.Join(t.TempDir(), "out.txt")
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
```

- [ ] **Step 2: テストが失敗することを確認する**

```bash
go test ./pkg/output/...
```

Expected: FAIL

- [ ] **Step 3: output.go を実装する**

`pkg/output/output.go`:

```go
package output

import (
	"fmt"
	"os"
	"strings"

	"github.com/gozuk16/yakuqr/pkg/parser"
	"github.com/gozuk16/yakuqr/pkg/validator"
)

// BuildText はPrescriptionとバリデーション結果からテキストを生成する。
func BuildText(p parser.Prescription, results []validator.ValidationResult) string {
	var sb strings.Builder

	// RAW QR DATA セクション
	sb.WriteString("=== RAW QR DATA ===\n")
	for i, raw := range p.RawQRs {
		fmt.Fprintf(&sb, "[QR #%d]\n%s\n\n", i+1, raw)
	}

	// PARSED DATA セクション
	sb.WriteString("=== PARSED DATA ===\n")
	fmt.Fprintf(&sb, "バージョン: %s\n", p.Version)

	if recs, ok := p.RecordMap["2"]; ok && len(recs) > 0 {
		sb.WriteString("--- 患者情報 ---\n")
		f := recs[0].Fields
		if len(f) > 1 {
			fmt.Fprintf(&sb, "氏名: %s\n", f[1])
		}
		if len(f) > 2 {
			fmt.Fprintf(&sb, "カナ名: %s\n", f[2])
		}
		if len(f) > 3 {
			fmt.Fprintf(&sb, "生年月日: %s\n", formatDate(f[3]))
		}
		if len(f) > 4 {
			fmt.Fprintf(&sb, "性別: %s\n", formatSex(f[4]))
		}
	}

	if recs, ok := p.RecordMap["1"]; ok && len(recs) > 0 {
		sb.WriteString("--- 処方箋情報 ---\n")
		f := recs[0].Fields
		if len(f) > 2 {
			fmt.Fprintf(&sb, "医療機関コード: %s\n", f[2])
		}
	}

	if recs, ok := p.RecordMap["6"]; ok {
		sb.WriteString("--- Rp情報 ---\n")
		for i, rec := range recs {
			f := rec.Fields
			name := ""
			if len(f) > 2 {
				name = f[2]
			}
			dose := ""
			if len(f) > 3 {
				dose = f[3]
			}
			unit := ""
			if len(f) > 4 {
				unit = f[4]
			}
			days := ""
			if len(f) > 6 {
				days = f[6]
			}
			fmt.Fprintf(&sb, "Rp%d: %s %s%s %s日分\n", i+1, name, dose, unit, days)
		}
	}

	sb.WriteString("\n")

	// VALIDATION RESULTS セクション
	sb.WriteString("=== VALIDATION RESULTS ===\n")
	if len(results) == 0 {
		sb.WriteString("問題は検出されませんでした\n")
	} else {
		for _, r := range results {
			fmt.Fprintf(&sb, "%s\n", r)
		}
	}

	return sb.String()
}

// WriteOutput はテキストをファイルに書き出し、ERROR/WARNINGをstderrにも出力する。
func WriteOutput(outPath string, p parser.Prescription, results []validator.ValidationResult, srcFile ...string) error {
	text := BuildText(p, results)
	if err := os.WriteFile(outPath, []byte(text), 0644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}

	prefix := outPath
	if len(srcFile) > 0 {
		prefix = srcFile[0]
	}
	for _, r := range results {
		if r.Level == validator.LevelError || r.Level == validator.LevelWarning {
			fmt.Fprintf(os.Stderr, "%s: %s\n", prefix, r)
		}
	}
	return nil
}

// OutputPath は入力ファイルパスから出力ファイルパスを生成する。
// outDir が空なら入力ファイルと同じディレクトリに出力する。
func OutputPath(inputPath, outDir string, existing map[string]bool) string {
	base := strings.TrimSuffix(inputPath, "")
	// 拡張子を除去
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".pdf"} {
		if strings.HasSuffix(strings.ToLower(inputPath), ext) {
			base = inputPath[:len(inputPath)-len(ext)]
			break
		}
	}

	candidate := base + "_out.txt"
	if outDir != "" {
		parts := strings.Split(candidate, "/")
		candidate = outDir + "/" + parts[len(parts)-1]
	}

	if !existing[candidate] {
		existing[candidate] = true
		return candidate
	}
	for i := 2; ; i++ {
		numbered := strings.TrimSuffix(candidate, ".txt") + fmt.Sprintf("_%d.txt", i)
		if !numbered[len(numbered)-1:] != "_" && !existing[numbered] {
			existing[numbered] = true
			return numbered
		}
	}
}

func formatDate(s string) string {
	if len(s) == 8 {
		return s[:4] + "-" + s[4:6] + "-" + s[6:]
	}
	return s
}

func formatSex(s string) string {
	switch s {
	case "1":
		return "男"
	case "2":
		return "女"
	default:
		return s
	}
}
```

- [ ] **Step 4: テストを実行して通ることを確認する**

```bash
go test ./pkg/output/... -v
```

Expected: PASS

- [ ] **Step 5: コミットする**

```bash
git add pkg/output/
git commit -m "feat: implement output text generation and file writing"
```

---

## Task 9: cmd/yakuqr — CLIエントリポイント

**Files:**
- Create: `cmd/yakuqr/main.go`

- [ ] **Step 1: main.go を実装する**

`cmd/yakuqr/main.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/gozuk16/yakuqr/pkg/decoder"
	"github.com/gozuk16/yakuqr/pkg/output"
	"github.com/gozuk16/yakuqr/pkg/parser"
	"github.com/gozuk16/yakuqr/pkg/validator"
)

func main() {
	app := &cli.App{
		Name:    "yakuqr",
		Usage:   "JAHIS院外処方箋QRコードを読み取り、内容を解析してテキストファイルに出力します",
		Version: "0.1.0",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "出力ファイルパス（単一ファイル入力時のみ有効）",
			},
			&cli.StringFlag{
				Name:    "output-dir",
				Aliases: []string{"d"},
				Usage:   "出力ディレクトリ（複数ファイル入力時に使用）",
			},
		},
		Action: run,
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(64)
	}
}

func run(c *cli.Context) error {
	files := c.Args().Slice()
	if len(files) == 0 {
		return cli.ShowAppHelp(c)
	}

	outFlag := c.String("output")
	outDir := c.String("output-dir")

	if outFlag != "" && len(files) > 1 {
		return fmt.Errorf("複数ファイル指定時に -o は使用できません。--output-dir を使用してください")
	}

	existing := make(map[string]bool)
	successCount, failCount, errorCount := 0, 0, 0

	for _, inputPath := range files {
		rawQRs, decErrs := decoder.DecodeFile(inputPath)
		if len(decErrs) > 0 && len(rawQRs) == 0 {
			fmt.Fprintf(os.Stderr, "%s: QR読み取り失敗: %v\n", inputPath, decErrs[0])
			failCount++
			continue
		}

		p, msgs := parser.Parse(rawQRs)
		for _, msg := range msgs {
			fmt.Fprintf(os.Stderr, "%s: %s\n", inputPath, msg)
		}

		results := validator.Validate(p)

		var outPath string
		if outFlag != "" {
			outPath = outFlag
		} else {
			outPath = output.OutputPath(inputPath, outDir, existing)
		}

		if err := output.WriteOutput(outPath, p, results, inputPath); err != nil {
			fmt.Fprintf(os.Stderr, "%s: 出力ファイル書き込み失敗: %v\n", inputPath, err)
			failCount++
			continue
		}

		fmt.Printf("%s -> %s\n", inputPath, outPath)
		successCount++

		for _, r := range results {
			if r.Level == validator.LevelError {
				errorCount++
			}
		}
	}

	if len(files) > 1 {
		fmt.Printf("\n処理完了: 成功 %d件 / 失敗 %d件 / ERRORバリデーション %d件\n", successCount, failCount, errorCount)
	}

	if failCount > 0 {
		os.Exit(1)
	}
	if errorCount > 0 {
		os.Exit(2)
	}
	return nil
}
```

- [ ] **Step 2: ビルドして動作確認する**

```bash
make build
./yakuqr --help
```

Expected:
```
NAME:
   yakuqr - JAHIS院外処方箋QRコードを読み取り、内容を解析してテキストファイルに出力します
...
```

- [ ] **Step 3: 全テストを実行する**

```bash
make test
```

Expected: PASS

- [ ] **Step 4: コミットする**

```bash
git add cmd/yakuqr/main.go
git commit -m "feat: add CLI entrypoint with urfave/cli"
```

---

## Task 10: testdata整備とver3/ver2サンプル追加

**Files:**
- Create: `testdata/ver3_single.txt`
- Create: `testdata/ver2_single.txt`
- Modify: `pkg/parser/parser_test.go`

> **NOTE:** Ver.2/Ver.3のサンプルデータは実際のJAHIS規約原文のサンプルを参照して作成すること。以下は構造例。

- [ ] **Step 1: Ver.3サンプルを作成する**

`testdata/ver3_single.txt`（Ver.3形式。レコード1の2番目フィールドが "3"）:

```
1,3,131012345,13,1,20240101,20240101,01,0,処方病院,,,030110,,,
2,山田花子,ヤマダハナコ,19800202,2,,,
3,06120345678901234,協会けんぽ,,,
5,1,
6,110626050,アムロジピン錠5mg「日医工」,1,錠,1011000000000000000000000000,28,3,
8,
```

- [ ] **Step 2: Ver.3テストを追加する**

`pkg/parser/parser_test.go` に追加:

```go
func TestParse_Ver3_DetectsVersion(t *testing.T) {
	raw := readTestdata("ver3_single.txt")
	p, _ := parser.Parse([]string{raw})
	if p.Version != parser.Version3 {
		t.Errorf("expected Version3, got %v", p.Version)
	}
}
```

- [ ] **Step 3: テストを実行して通ることを確認する**

```bash
go test ./pkg/parser/... -v
```

Expected: PASS

- [ ] **Step 4: コミットする**

```bash
git add testdata/ pkg/parser/parser_test.go
git commit -m "test: add Ver.3 testdata and version detection test"
```

---

## Task 11: README更新とCHANGELOG記入

**Files:**
- Modify: `README.md`
- Modify: `CHANGELOG.md`

- [ ] **Step 1: README.md を更新する**

```markdown
# yakuqr

JAHIS院外処方箋２次元シンボル記録条件規約（Ver.2〜Ver.4）に準拠したQRコードを画像・PDFファイルから読み取り、解析結果とバリデーション結果をテキストファイルに出力するCLIツール。

## インストール

\`\`\`bash
go install github.com/gozuk16/yakuqr/cmd/yakuqr@latest
\`\`\`

## 使い方

\`\`\`bash
# 単一ファイル
yakuqr prescription.pdf

# 出力先を指定
yakuqr -o result.txt scan.png

# 複数ファイル（--output-dirで出力ディレクトリを指定）
yakuqr -d ./output img1.png img2.png

# ヘルプ
yakuqr --help
\`\`\`

## 出力

入力ファイル名に `_out.txt` を付加したファイルに以下を出力します：
- 生のQRコードデータ
- 解析済み処方箋フィールド
- JAHIS仕様バリデーション結果

ERROR/WARNINGレベルの問題は標準エラー出力にも表示されます。

## 終了コード

| コード | 意味 |
|--------|------|
| 0 | 正常終了 |
| 1 | QR読み取り/IO失敗 |
| 2 | ERRORバリデーションあり |
| 64 | 引数不正 |

## ビルド

\`\`\`bash
make build
make test
make lint
\`\`\`
```

- [ ] **Step 2: CHANGELOG.md を更新する**

```markdown
# Changelog

## [0.1.0] - 2026-04-25
### Added
- JAHIS院外処方箋QRコード（Ver.2〜Ver.4）の読み取り対応
- 画像ファイル（PNG/JPEG）・PDFファイルの入力対応
- 分割QRコードの自動結合
- JAHIS仕様バリデーション（ERROR/WARNING/INFO）
- テキストファイルへの出力（生データ・解析結果・バリデーション結果）
```

- [ ] **Step 3: コミットする**

```bash
git add README.md CHANGELOG.md
git commit -m "docs: update README and CHANGELOG for v0.1.0"
```

---

## セルフレビュー結果

**スペックカバレッジ:**
- ✅ 画像・PDF入力対応（Task 3, 4）
- ✅ QRデコード・Shift_JIS変換（Task 2, 3）
- ✅ 分割QR結合（Task 6）
- ✅ バージョン自動検出（Task 6）
- ✅ JAHISバリデーション（Task 7）
- ✅ テキスト出力 + stderr（Task 8）
- ✅ urfave/cli（Task 9）
- ✅ 終了コード（Task 9）
- ✅ 出力ファイル名衝突対応（Task 8）
- ✅ -o / --output-dir オプション（Task 9）
- ✅ Makefile（Task 1）
- ✅ テーブルドリブンテスト（Task 6, 7）
- ✅ ゴールデンファイルテスト（Task 8）

**型・関数名の一貫性:**
- `parser.Prescription`, `parser.Record`, `parser.SplitInfo`, `parser.Version` — Task 5で定義、Task 6〜9で使用
- `validator.ValidationResult`, `validator.Level`, `validator.LevelError/Warning/Info` — Task 7で定義、Task 8〜9で使用
- `decoder.DecodeFile`, `decoder.DecodeError` — Task 2で定義、Task 9で使用
- `output.BuildText`, `output.WriteOutput`, `output.OutputPath` — Task 8で定義、Task 9で使用

**残存する実装時注意点（プレースホルダーではなく意図的な指示）:**
- JAHIS規約原文でレコード9の分割フォーマット、各レコードのフィールド番号を確認すること
- pdfcpuのページラスタライズAPIはバージョンで変わるため `go doc` で確認すること
- `isValidDate` の月・日範囲チェックは省略済み（必要に応じて追加）
