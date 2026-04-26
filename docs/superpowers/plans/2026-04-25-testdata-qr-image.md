# テストデータ QR画像管理 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 既存 testdata/*.txt から QR コード画像を自動生成して `testdata/generated/` にコミットし、デコーダー統合テストとパーサー E2E パイプラインテストを追加する。インターネット収集サンプルは `testdata/collected/` ディレクトリに手動配置する仕組みを整える。

**Architecture:** `tools/gen-testdata-qr/main.go` が `testdata/*.txt` を Shift_JIS エンコードして QR 画像 PNG を生成し `testdata/generated/` に置く。`pkg/decoder/decoder_test.go` がこれを使ってデコーダー統合テストを常時実行し、`pkg/parser/integration_test.go` が decoder → parser → validator の全パイプラインを検証する。収集サンプルは `.gitignore` 対象の `testdata/collected/` に置き、存在するときだけテストが走る。

**Tech Stack:** Go 1.25.5、`github.com/skip2/go-qrcode`（QR生成）、`golang.org/x/text`（Shift_JIS 変換、既存）、`github.com/makiuchi-d/gozxing`（QR読み取り、既存）

---

## ファイル構成

| 操作 | パス | 内容 |
|------|------|------|
| 作成 | `tools/gen-testdata-qr/main.go` | QR画像生成ツール |
| 作成 | `testdata/generated/README.md` | 再生成手順 |
| 作成（生成） | `testdata/generated/qr_ver2_single.png` | ver2_single.txt から生成 |
| 作成（生成） | `testdata/generated/qr_ver3_single.png` | ver3_single.txt から生成 |
| 作成（生成） | `testdata/generated/qr_ver4_single.png` | ver4_single.txt から生成 |
| 作成（生成） | `testdata/generated/qr_ver4_split_1.png` | ver4_split_1.txt から生成 |
| 作成（生成） | `testdata/generated/qr_ver4_split_2.png` | ver4_split_2.txt から生成 |
| 作成 | `testdata/collected/README.md` | ITmedia サンプルのDL手順 |
| 作成 | `testdata/collected/.gitkeep` | ディレクトリ維持用 |
| 修正 | `pkg/decoder/decoder_test.go` | 統合テスト追加、古いスキップテスト削除 |
| 作成 | `pkg/parser/integration_test.go` | E2E パイプラインテスト |
| 修正 | `.gitignore` | collected/*.jpg 等を除外 |
| 修正 | `Makefile` | gen-testdata ターゲット追加 |

---

### Task 1: go-qrcode 依存を追加する

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: go get で依存を追加**

```bash
go get github.com/skip2/go-qrcode@latest
```

Expected output（バージョンは最新に変わる可能性あり）:
```
go: added github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
```

- [ ] **Step 2: go.mod に追記されたことを確認**

```bash
grep go-qrcode go.mod
```

Expected:
```
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
```

---

### Task 2: QR画像生成ツールを実装する

**Files:**
- Create: `tools/gen-testdata-qr/main.go`

- [ ] **Step 1: ディレクトリを作成**

```bash
mkdir -p tools/gen-testdata-qr
```

- [ ] **Step 2: main.go を作成**

`tools/gen-testdata-qr/main.go`:

```go
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
```

- [ ] **Step 3: ツールが構文エラーなくビルドできることを確認**

```bash
go build ./tools/gen-testdata-qr/
```

Expected: エラーなし（バイナリは生成しない）

---

### Task 3: QR画像を生成して generated/README.md を作成する

**Files:**
- Create: `testdata/generated/*.png`（5ファイル）
- Create: `testdata/generated/README.md`

- [ ] **Step 1: QR画像を生成**

```bash
go run ./tools/gen-testdata-qr/
```

Expected output:
```
generated: /path/to/testdata/generated/qr_ver2_single.png
generated: /path/to/testdata/generated/qr_ver3_single.png
generated: /path/to/testdata/generated/qr_ver4_single.png
generated: /path/to/testdata/generated/qr_ver4_split_1.png
generated: /path/to/testdata/generated/qr_ver4_split_2.png
generated: /path/to/testdata/generated/qr_ver4_split_only2.png
```

- [ ] **Step 2: 生成ファイルを確認**

```bash
ls testdata/generated/
```

Expected: `qr_ver2_single.png  qr_ver3_single.png  qr_ver4_single.png  qr_ver4_split_1.png  qr_ver4_split_2.png  qr_ver4_split_only2.png` が存在すること

- [ ] **Step 3: generated/README.md を作成**

`testdata/generated/README.md`:

```markdown
# testdata/generated/

このディレクトリには `testdata/*.txt` から自動生成した JAHIS 形式 QR コード画像を格納しています。

## 再生成手順

リポジトリルートで以下を実行してください:

```bash
make gen-testdata
# または
go run ./tools/gen-testdata-qr/
```

## ファイル一覧

| ファイル | 元データ | 説明 |
|---------|---------|------|
| qr_ver2_single.png | ver2_single.txt | JAHIS Ver.2 単一QR |
| qr_ver3_single.png | ver3_single.txt | JAHIS Ver.3 単一QR |
| qr_ver4_single.png | ver4_single.txt | JAHIS Ver.4 単一QR（JAHISTC04形式） |
| qr_ver4_split_1.png | ver4_split_1.txt | JAHIS Ver.4 分割QR その1 |
| qr_ver4_split_2.png | ver4_split_2.txt | JAHIS Ver.4 分割QR その2 |
| qr_ver4_split_only2.png | ver4_split_only2.txt | 分割QR欠落テスト用（その2のみ） |

## 注意

- エンコーディングは Shift_JIS（実際の JAHIS QR コードに準拠）
- サイズ: 256×256 px、誤り訂正レベル M
- これらのファイルはリポジトリにコミットされます
```

- [ ] **Step 4: 最初のコミット**

```bash
git add tools/gen-testdata-qr/ testdata/generated/
git commit -m "feat: QR画像生成ツールとtestdata/generated/を追加"
```

---

### Task 4: .gitignore と Makefile を更新する

**Files:**
- Modify: `.gitignore`
- Modify: `Makefile`

- [ ] **Step 1: .gitignore に収集サンプルを除外するパターンを追加**

`.gitignore` の末尾に追記:

```
# Collected test samples (copyrighted, not committed)
testdata/collected/*.jpg
testdata/collected/*.jpeg
testdata/collected/*.png
testdata/collected/*.pdf
```

- [ ] **Step 2: Makefile に gen-testdata ターゲットを追加**

`Makefile` を以下に更新:

```makefile
BINARY := yakuqr
CMD := ./cmd/yakuqr

.PHONY: build test lint clean gen-testdata

build:
	go build -o $(BINARY) $(CMD)

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY)

gen-testdata:
	go run ./tools/gen-testdata-qr/
```

- [ ] **Step 3: コミット**

```bash
git add .gitignore Makefile
git commit -m "chore: .gitignore に collected/ サンプル除外を追加、Makefile に gen-testdata ターゲットを追加"
```

---

### Task 5: デコーダーテストをテーブル駆動に更新する

**Files:**
- Modify: `pkg/decoder/decoder_test.go`

- [ ] **Step 1: decoder_test.go 全体を書き換える**

`pkg/decoder/decoder_test.go`:

```go
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
		tc := tc
		t.Run(tc.file, func(t *testing.T) {
			path := testdataPath(filepath.Join("generated", tc.file))
			results, errs := decoder.DecodeFile(path)
			if len(errs) > 0 {
				t.Fatalf("decode errors: %v", errs)
			}
			if len(results) == 0 {
				t.Fatal("no QR decoded")
			}
			if !strings.Contains(results[0], tc.wantSubstr) {
				t.Errorf("expected %q in decoded result, got: %q", tc.wantSubstr, results[0])
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
		f := f
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
```

- [ ] **Step 2: テストを実行して generated/ テストが失敗することを確認（まだ統合テストを書いていないため、ここで失敗は正常）**

```bash
go test ./pkg/decoder/ -v -run TestDecodeImage_Generated 2>&1 | head -20
```

Expected: FAIL（generated/*.png が存在しない、または `os.Stat` でエラー）— Task 3 でファイルを生成済みであれば PASS になる

- [ ] **Step 3: コミット**

```bash
git add pkg/decoder/decoder_test.go
git commit -m "test: デコーダーテストをテーブル駆動に更新し generated/collected 対応を追加"
```

---

### Task 6: パーサー E2E 統合テストを作成する

**Files:**
- Create: `pkg/parser/integration_test.go`

- [ ] **Step 1: integration_test.go を作成**

`pkg/parser/integration_test.go`:

```go
package parser_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gozuk16/yakuqr/pkg/decoder"
	"github.com/gozuk16/yakuqr/pkg/parser"
	"github.com/gozuk16/yakuqr/pkg/validator"
)

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
	{"qr_ver4_single.png", parser.Version4, "1"},
	{"qr_ver2_single.png", parser.Version2, "2"},
	{"qr_ver3_single.png", parser.Version3, "2"},
}

func TestPipeline_DecodeParseValidate(t *testing.T) {
	for _, tc := range pipelineCases {
		tc := tc
		t.Run(tc.imageFile, func(t *testing.T) {
			path := generatedImagePath(tc.imageFile)

			// Step 1: decode
			rawQRs, errs := decoder.DecodeFile(path)
			if len(errs) > 0 {
				t.Fatalf("decode errors: %v", errs)
			}
			if len(rawQRs) == 0 {
				t.Fatal("no QR decoded")
			}

			// Step 2: parse
			p, _ := parser.Parse(rawQRs)
			if p.Version != tc.wantVersion {
				t.Errorf("version: want %v, got %v", tc.wantVersion, p.Version)
			}
			if _, ok := p.RecordMap[tc.wantRecord]; !ok {
				t.Errorf("RecordMap[%q] not found", tc.wantRecord)
			}

			// Step 3: validate — Ver.2/Ver.3 は INFO レベルのみ許容
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

	// 2枚それぞれからQRテキストを取得
	qrs1, errs1 := decoder.DecodeFile(path1)
	if len(errs1) > 0 {
		t.Fatalf("decode split_1 errors: %v", errs1)
	}
	qrs2, errs2 := decoder.DecodeFile(path2)
	if len(errs2) > 0 {
		t.Fatalf("decode split_2 errors: %v", errs2)
	}

	allQRs := append(qrs1, qrs2...)

	p, msgs := parser.Parse(allQRs)

	// 警告がないことを確認（分割QRが正しく結合されているはず）
	for _, m := range msgs {
		t.Logf("parse message: %s", m)
	}

	if p.Version != parser.Version4 {
		t.Errorf("version: want Version4, got %v", p.Version)
	}
	if _, ok := p.RecordMap["201"]; !ok {
		t.Error("RecordMap[\"201\"] not found — split QR combination may have failed")
	}

	// バリデーションエラーがないことを確認
	results := validator.Validate(p)
	for _, r := range results {
		if r.Level == validator.LevelError {
			t.Errorf("unexpected validation ERROR: %v", r)
		}
	}
}
```

- [ ] **Step 2: テストを実行して PASS することを確認**

```bash
go test ./pkg/parser/ -v -run TestPipeline 2>&1
```

Expected: すべて PASS

- [ ] **Step 3: 全テストが引き続き PASS することを確認**

```bash
go test ./...
```

Expected: 全パッケージ PASS

- [ ] **Step 4: コミット**

```bash
git add pkg/parser/integration_test.go
git commit -m "test: decoder→parser→validator の E2E パイプラインテストを追加"
```

---

### Task 7: testdata/collected/ を整備する

**Files:**
- Create: `testdata/collected/README.md`
- Create: `testdata/collected/.gitkeep`

- [ ] **Step 1: ディレクトリを作成して .gitkeep を置く**

```bash
mkdir -p testdata/collected
touch testdata/collected/.gitkeep
```

- [ ] **Step 2: collected/README.md を作成**

`testdata/collected/README.md`:

```markdown
# testdata/collected/

このディレクトリには、インターネットから収集した実世界の JAHIS 処方箋 QR コードサンプルを格納します。

**著作権の関係上、画像ファイルはリポジトリにコミットしません。**
テストを実行する前に、以下の手順で手動ダウンロードして配置してください。

---

## ITmedia サンプル画像（架空データ使用）

出典記事: https://nlab.itmedia.co.jp/cont/articles/3293608/

著作権: Copyright © ITmedia Inc. All Rights Reserved.
利用条件: テスト目的の個人利用のみ。再配布・リポジトリへのコミット不可。

### ダウンロード手順

```bash
cd testdata/collected/

curl -O "https://preresearch.image.itmedia.co.jp/nl/articles/1907/25/miya_1907okusuriqr01.jpg"
curl -O "https://preresearch.image.itmedia.co.jp/nl/articles/1907/25/miya_1907okusuriqr02.jpg"
curl -O "https://preresearch.image.itmedia.co.jp/nl/articles/1907/25/miya_1907okusuriqr03.jpg"
```

ダウンロード後、`go test ./pkg/decoder/ -run TestDecodeImage_Collected` でテストが実行されます。

---

## サンプルを追加する場合

1. 著作権・利用条件を確認し、テスト目的での利用が許可されているもののみ追加してください
2. このREADMEにファイル名・出典URL・著作権情報・取得日を記載してください
3. 画像ファイルは `.gitignore` 対象のため、コミットしないでください
```

- [ ] **Step 3: コミット**

```bash
git add testdata/collected/
git commit -m "chore: testdata/collected/ ディレクトリとダウンロード手順を追加"
```

---

### Task 8: 最終確認とまとめコミット

**Files:**
- Verify: 全テストが PASS

- [ ] **Step 1: 全テストが PASS することを確認**

```bash
go test ./... -v 2>&1 | grep -E "^(ok|FAIL|---)"
```

Expected:
```
ok  	github.com/gozuk16/yakuqr/pkg/decoder
ok  	github.com/gozuk16/yakuqr/pkg/output
ok  	github.com/gozuk16/yakuqr/pkg/parser
ok  	github.com/gozuk16/yakuqr/pkg/validator
```

- [ ] **Step 2: generated テストが実際に実行されている（スキップでない）ことを確認**

```bash
go test ./pkg/decoder/ -v -run TestDecodeImage_Generated 2>&1
```

Expected: `--- PASS: TestDecodeImage_Generated/qr_ver4_single.png` 等が表示される（SKIP でない）

- [ ] **Step 3: CHANGELOG を更新**

`CHANGELOG.md` の `[0.1.1]` セクションの上に追加:

```markdown
## [0.1.2] - 2026-04-25
### Added
- `testdata/generated/` に QR 画像テストデータ（go-qrcode で自動生成）を追加
- `pkg/decoder/decoder_test.go` にテーブル駆動デコーダー統合テストを追加
- `pkg/parser/integration_test.go` に decoder→parser→validator E2E パイプラインテストを追加
- `testdata/collected/README.md` に ITmedia サンプルの手動ダウンロード手順を記載
- `make gen-testdata` で QR 画像を再生成できる Makefile ターゲットを追加
```

- [ ] **Step 4: 最終コミット**

```bash
git add CHANGELOG.md
git commit -m "chore: CHANGELOG 更新"
```
