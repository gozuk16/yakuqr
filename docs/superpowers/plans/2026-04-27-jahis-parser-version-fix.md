# JAHISパーサー バージョン検出修正 実装計画

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** JAHIS電子版お薬手帳データフォーマット仕様書 Ver.1.0〜Ver.2.6（JAHISTC01〜09）の全バージョンに対応し、バージョン未検出時はフォールバックせずエラーを返すよう Go パーサーを修正する。

**Architecture:** `types.go` はユーザーが更新済み（`Version1_0`〜`Version2_6`）だが、`parser.go`/`rules.go`/テストが古い定数（`Version2/3/4`）を参照しており現在コンパイルエラー。まず全ファイルの定数参照を修正してコンパイルを通し、次に新バージョン対応のテストを追加しながら実装を更新する。

**Tech Stack:** Go 1.22、標準ライブラリのみ（`strings`, `strconv`）

---

## ファイル構成

```
pkg/parser/types.go           ← Version.String() に欠落ケース追加（Modify）
pkg/parser/parser.go          ← detectVersion 全面修正（Modify）
pkg/parser/parser_test.go     ← 定数更新 + JAHISTC01/08/09/1桁/不明テスト追加（Modify）
pkg/parser/integration_test.go← 定数更新（Modify）
pkg/validator/rules.go        ← rulesFor に VersionUnknown 分岐追加（Modify）
pkg/validator/validator_test.go← 定数更新 + VersionUnknown テスト追加（Modify）
pkg/output/output.go          ← BuildText にバージョン分岐追加（Modify）
pkg/output/output_test.go     ← 定数更新 + バージョン分岐テスト追加（Modify）
testdata/ver1_0_single.txt    ← 新規（JAHISTC01形式）
testdata/ver1_0_1digit.txt    ← 新規（JAHISTC1形式、1桁）
testdata/ver2_5_single.txt    ← 新規（JAHISTC08形式）
testdata/ver2_6_single.txt    ← 新規（JAHISTC09形式）
docs/superpowers/specs/2026-04-25-jahis-qr-cli-design.md ← バージョン表更新（Modify）
```

---

### Task 1: テストデータ追加

**Files:**
- Create: `testdata/ver1_0_single.txt`
- Create: `testdata/ver1_0_1digit.txt`
- Create: `testdata/ver2_5_single.txt`
- Create: `testdata/ver2_6_single.txt`

背景: JAHISTC01〜03は旧フォーマット（レコード2=患者情報, レコード6=薬品情報）。JAHISTC04以降は新フォーマット（レコード1=患者情報, レコード201=薬品情報）。

- [ ] **Step 1: JAHISTC01形式（2桁）のテストデータを作成**

`testdata/ver1_0_single.txt` の内容:

```
JAHISTC01,1
1,1,131012345,13,1,20240101,20240101,01,0,処方病院,,,030110,,,
2,山田一郎,ヤマダイチロウ,19600101,1,,,
3,06120345678901234,協会けんぽ,,,
5,1,
6,110626050,アムロジピン錠5mg「日医工」,1,錠,1011000000000000000000000000,28,3,
8,
```

- [ ] **Step 2: JAHISTC1形式（1桁）のテストデータを作成**

`testdata/ver1_0_1digit.txt` の内容（ヘッダーの数字が1桁）:

```
JAHISTC1,1
1,1,131012345,13,1,20240101,20240101,01,0,処方病院,,,030110,,,
2,山田一郎,ヤマダイチロウ,19600101,1,,,
3,06120345678901234,協会けんぽ,,,
5,1,
6,110626050,アムロジピン錠5mg「日医工」,1,錠,1011000000000000000000000000,28,3,
8,
```

- [ ] **Step 3: JAHISTC08形式（Ver.2.5）のテストデータを作成**

`testdata/ver2_5_single.txt` の内容（新フォーマット、レコード1=患者情報）:

```
JAHISTC08,1
1,山田五郎,1,19500501,100-0001,東京都千代田区千代田1-1,,,0,テスト
5,20240101,1
11,薬局テスト,13,4,1234567,1000001,東京都千代田区1-1,03-1234-5678,1
15,医師テスト,,1
201,1,アムロジピン錠5mg「日医工」,1,錠,4,1149019F1625,1
301,1,（1日1回　朝食後服用）,28,日分,1,1,,1
```

- [ ] **Step 4: JAHISTC09形式（Ver.2.6）のテストデータを作成**

`testdata/ver2_6_single.txt` の内容（JAHISTC09ヘッダーのみ異なる）:

```
JAHISTC09,1
1,山田六郎,1,19600601,100-0001,東京都千代田区千代田1-1,,,0,テスト
5,20240101,1
11,薬局テスト,13,4,1234567,1000001,東京都千代田区1-1,03-1234-5678,1
15,医師テスト,,1
201,1,アムロジピン錠5mg「日医工」,1,錠,4,1149019F1625,1
301,1,（1日1回　朝食後服用）,28,日分,1,1,,1
```

- [ ] **Step 5: コミット**

```bash
git add testdata/ver1_0_single.txt testdata/ver1_0_1digit.txt testdata/ver2_5_single.txt testdata/ver2_6_single.txt
git commit -m "test: JAHISTC01/1桁/08/09形式のテストデータを追加"
```

---

### Task 2: `types.go` — `Version.String()` 補完

**Files:**
- Modify: `pkg/parser/types.go:19-36`

現在 `Version1_0`, `Version1_1`, `Version2_0`, `VersionUnknown` が `String()` に欠落している。

- [ ] **Step 1: `Version.String()` の全ケースを確認**

```bash
grep -n "case Version" pkg/parser/types.go
```

Expected: `Version2_1` から `Version2_6` の6ケースのみ存在する。

- [ ] **Step 2: 欠落ケースを追加**

`pkg/parser/types.go` の `String()` メソッドを以下に置き換える:

```go
func (v Version) String() string {
	switch v {
	case Version1_0:
		return "Ver.1.0"
	case Version1_1:
		return "Ver.1.1"
	case Version2_0:
		return "Ver.2.0"
	case Version2_1:
		return "Ver.2.1"
	case Version2_2:
		return "Ver.2.2"
	case Version2_3:
		return "Ver.2.3"
	case Version2_4:
		return "Ver.2.4"
	case Version2_5:
		return "Ver.2.5"
	case Version2_6:
		return "Ver.2.6"
	default:
		return "Unknown"
	}
}
```

- [ ] **Step 3: コンパイル確認**

```bash
go build ./pkg/parser/...
```

Expected: `pkg/parser/parser.go` のコンパイルエラーのみ残る（types.go自体はエラーなし）。

- [ ] **Step 4: コミット**

```bash
git add pkg/parser/types.go
git commit -m "fix: Version.String() に Ver.1.0/1.1/2.0/Unknown ケースを追加"
```

---

### Task 3: `parser.go` — `detectVersion` 修正

**Files:**
- Modify: `pkg/parser/parser.go:126-161`

- [ ] **Step 1: 現在のコンパイルエラーを確認**

```bash
go build ./... 2>&1
```

Expected:
```
pkg/parser/parser.go:139:12: undefined: Version2
pkg/parser/parser.go:141:12: undefined: Version3
pkg/parser/parser.go:143:12: undefined: Version4
pkg/parser/parser.go:152:12: undefined: Version2
pkg/parser/parser.go:154:12: undefined: Version3
pkg/parser/parser.go:156:12: undefined: Version4
pkg/parser/parser.go:160:9: undefined: Version4
```

- [ ] **Step 2: `detectVersion` 関数を全面置き換え**

`pkg/parser/parser.go` の `detectVersion` 関数（126〜161行）を以下に置き換える:

```go
// detectVersion はJAHISバージョンを検出する。
// JAHISTC ヘッダーがあれば番号から直接 Version(n) を返す（1桁・2桁どちらも対応）。
// ヘッダーなしの旧フォーマットはレコード種別1のフィールド2（バージョン番号文字）を参照する。
// どちらでも検出できない場合は VersionUnknown と ERROR メッセージを返す。
func detectVersion(rm map[string][]Record, rawQRs []string) (Version, string) {
	for _, raw := range rawQRs {
		lines := splitLines(raw)
		if len(lines) == 0 {
			continue
		}
		first := strings.TrimSpace(lines[0])
		if strings.HasPrefix(first, "JAHISTC") {
			header := first
			if i := strings.Index(first, ","); i >= 0 {
				header = first[:i]
			}
			numStr := strings.TrimLeft(header[7:], "0")
			if n, err := strconv.Atoi(numStr); err == nil && n >= 1 && n <= 9 {
				return Version(n), ""
			}
		}
	}
	// ヘッダーなし旧フォーマット: レコード種別1のフィールド[1]がバージョン番号文字列
	if recs, ok := rm["1"]; ok && len(recs) > 0 && len(recs[0].Fields) >= 2 {
		switch recs[0].Fields[1] {
		case "1":
			return Version1_0, ""
		case "2":
			return Version1_1, ""
		case "3":
			return Version2_0, ""
		}
	}
	return VersionUnknown, "[ERROR] バージョンを検出できませんでした"
}
```

- [ ] **Step 3: コンパイル確認**

```bash
go build ./...
```

Expected: エラーなし（テストはまだ古い定数を参照しているが build は通る）。

- [ ] **Step 4: コミット**

```bash
git add pkg/parser/parser.go
git commit -m "fix: detectVersion を全JAHISTC番号対応・1桁対応・未検出時VersionUnknownに修正"
```

---

### Task 4: `parser_test.go` — 定数更新と新バージョンテスト追加

**Files:**
- Modify: `pkg/parser/parser_test.go`

- [ ] **Step 1: 古い定数参照を新しい定数に置き換え**

`pkg/parser/parser_test.go` の古い定数を以下の通り置き換える:

| 旧 | 新 |
|---|---|
| `parser.Version4` | `parser.Version2_1` |
| `parser.Version3` | `parser.Version2_0` |
| `parser.Version2` | `parser.Version1_1` |

`TestParse_SingleQR_Version`（25〜28行）:
```go
func TestParse_SingleQR_Version(t *testing.T) {
	raw := readTestdata("ver4_single.txt")
	p, _ := parser.Parse([]string{raw})
	if p.Version != parser.Version2_1 {
		t.Errorf("expected Version2_1, got %v", p.Version)
	}
}
```

`TestParse_SplitQR_Combined`（81〜83行の `Version4` 参照）:
```go
	if p.Version != parser.Version2_1 {
		t.Errorf("expected Version2_1 after split QR combine, got %v", p.Version)
	}
```

`TestParse_Ver3_DetectsVersion`（71〜76行）:
```go
func TestParse_Ver3_DetectsVersion(t *testing.T) {
	raw := readTestdata("ver3_single.txt")
	p, _ := parser.Parse([]string{raw})
	if p.Version != parser.Version2_0 {
		t.Errorf("expected Version2_0, got %v", p.Version)
	}
}
```

`TestParse_Ver2_DetectsVersion`（79〜85行）:
```go
func TestParse_Ver2_DetectsVersion(t *testing.T) {
	raw := readTestdata("ver2_single.txt")
	p, _ := parser.Parse([]string{raw})
	if p.Version != parser.Version1_1 {
		t.Errorf("expected Version1_1, got %v", p.Version)
	}
}
```

`TestParse_VersionUnknown_FallsBackToVer4`（87〜101行）をリネームして内容を更新:
```go
func TestParse_VersionUnknown_ReturnsError(t *testing.T) {
	raw := "99,unknown\n2,テスト,テスト,19900101,1,,,"
	p, msgs := parser.Parse([]string{raw})
	if p.Version != parser.VersionUnknown {
		t.Errorf("expected VersionUnknown, got %v", p.Version)
	}
	found := false
	for _, msg := range msgs {
		if strings.Contains(msg, "[ERROR]") && strings.Contains(msg, "バージョン") {
			found = true
		}
	}
	if !found {
		t.Error("expected ERROR message about version detection failure")
	}
}
```

- [ ] **Step 2: 既存テストが通ることを確認**

```bash
go test ./pkg/parser/... -run "TestParse_SingleQR|TestParse_SplitQR|TestParse_Ver" -v
```

Expected: 全テスト PASS。

- [ ] **Step 3: 新バージョン対応テストを追加**

`pkg/parser/parser_test.go` の末尾に追加:

```go
func TestParse_JAHISTC01_DetectsVersion1_0(t *testing.T) {
	raw := readTestdata("ver1_0_single.txt")
	p, _ := parser.Parse([]string{raw})
	if p.Version != parser.Version1_0 {
		t.Errorf("expected Version1_0, got %v", p.Version)
	}
}

func TestParse_JAHISTC1_1digit_DetectsVersion1_0(t *testing.T) {
	raw := readTestdata("ver1_0_1digit.txt")
	p, _ := parser.Parse([]string{raw})
	if p.Version != parser.Version1_0 {
		t.Errorf("expected Version1_0 from 1-digit JAHISTC header, got %v", p.Version)
	}
}

func TestParse_JAHISTC08_DetectsVersion2_5(t *testing.T) {
	raw := readTestdata("ver2_5_single.txt")
	p, _ := parser.Parse([]string{raw})
	if p.Version != parser.Version2_5 {
		t.Errorf("expected Version2_5, got %v", p.Version)
	}
}

func TestParse_JAHISTC09_DetectsVersion2_6(t *testing.T) {
	raw := readTestdata("ver2_6_single.txt")
	p, _ := parser.Parse([]string{raw})
	if p.Version != parser.Version2_6 {
		t.Errorf("expected Version2_6, got %v", p.Version)
	}
}
```

- [ ] **Step 4: 全パーサーテストを実行**

```bash
go test ./pkg/parser/... -v -run "^TestParse_"
```

Expected: 全テスト PASS。

- [ ] **Step 5: コミット**

```bash
git add pkg/parser/parser_test.go
git commit -m "test: parser_test の定数を更新し JAHISTC01/1桁/08/09 バージョン検出テストを追加"
```

---

### Task 5: `rules.go` — `VersionUnknown` 分岐追加

**Files:**
- Modify: `pkg/validator/rules.go`

- [ ] **Step 1: 失敗するテストを書く**

`pkg/validator/validator_test.go` の末尾に追加:

```go
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
```

`validator_test.go` の先頭 import に `"strings"` が必要。現在のimportを確認して追加:
```go
import (
	"strings"
	"testing"

	"github.com/gozuk16/yakuqr/pkg/parser"
	"github.com/gozuk16/yakuqr/pkg/validator"
)
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test ./pkg/validator/... -run "TestValidate_UnknownVersion" -v
```

Expected: FAIL（現在 `rulesFor` に `VersionUnknown` ケースがない）。

- [ ] **Step 3: `validator_test.go` の古い定数を更新**

| 旧 | 新 |
|---|---|
| `parser.Version4` | `parser.Version2_1` |
| `parser.Version2` | `parser.Version1_1` |

`makeMinimalPrescriptionVer4`（21行）:
```go
return parser.Prescription{Version: parser.Version2_1, Records: records, RecordMap: rm}
```

`makeMinimalPrescriptionVer2`（37行）:
```go
return parser.Prescription{Version: parser.Version1_1, Records: records, RecordMap: rm}
```

`ver2ver3Rules` 末尾の INFO ルール条件（`rules.go` 150行）を後で更新する。

- [ ] **Step 4: `rules.go` の `rulesFor` を修正**

`pkg/validator/rules.go` の `rulesFor` 関数（11〜16行）を以下に置き換え:

```go
func rulesFor(v parser.Version) []rule {
	if v == parser.VersionUnknown {
		return unknownVersionRules()
	}
	if v >= parser.Version2_1 {
		return ver4Rules()
	}
	return ver2ver3Rules(v)
}

func unknownVersionRules() []rule {
	return []rule{{
		field: "バージョン",
		level: LevelError,
		check: func(p parser.Prescription) (bool, string) {
			return false, "バージョンを検出できませんでした。QRコードの形式を確認してください"
		},
	}}
}
```

また `ver2ver3Rules` の末尾にあるバージョン互換性 INFO ルール（150〜158行）の条件を修正:

```go
	if v <= parser.Version2_0 {
		base = append(base, rule{
			field: "バージョン互換性",
			level: LevelInfo,
			check: func(p parser.Prescription) (bool, string) {
				return false, "JAHISTC01〜03形式（Ver.1.0〜2.0）を検出しました。一部のフィールドはVer.2.1以降と異なる場合があります"
			},
		})
	}
```

- [ ] **Step 5: 全バリデーターテストを実行**

```bash
go test ./pkg/validator/... -v
```

Expected: 全テスト PASS。

- [ ] **Step 6: コミット**

```bash
git add pkg/validator/rules.go pkg/validator/validator_test.go
git commit -m "fix: validator に VersionUnknown エラールールを追加し定数を更新"
```

---

### Task 6: `output.go` — バージョン分岐追加

**Files:**
- Modify: `pkg/output/output.go:23-69`
- Modify: `pkg/output/output_test.go`

- [ ] **Step 1: 失敗するテストを書く**

`pkg/output/output_test.go` に以下を追加:

```go
func makeTestPrescriptionVer2_1() (parser.Prescription, []validator.ValidationResult) {
	records := []parser.Record{
		{Type: "1", Fields: []string{"1", "山田太郎", "1", "19700101", "100-0001"}},
		{Type: "201", Fields: []string{"201", "1", "アムロジピン錠5mg", "1", "錠"}},
	}
	rm := map[string][]parser.Record{
		"1":   {records[0]},
		"201": {records[1]},
	}
	p := parser.Prescription{
		Version:   parser.Version2_1,
		RawQRs:    []string{"JAHISTC04,1\n1,山田太郎,1,19700101"},
		Records:   records,
		RecordMap: rm,
	}
	return p, nil
}

func TestBuildText_Ver2_1_ShowsRecord1AsPatient(t *testing.T) {
	p, results := makeTestPrescriptionVer2_1()
	text := output.BuildText(p, results)
	if !strings.Contains(text, "山田太郎") {
		t.Error("expected patient name from record 1 in Ver.2.1 output")
	}
}

func makeTestPrescriptionUnknown() (parser.Prescription, []validator.ValidationResult) {
	p := parser.Prescription{
		Version:   parser.VersionUnknown,
		RawQRs:    []string{"99,unknown"},
		Records:   []parser.Record{},
		RecordMap: map[string][]parser.Record{},
	}
	results := []validator.ValidationResult{
		{Level: validator.LevelError, Field: "バージョン", Message: "バージョンを検出できませんでした"},
	}
	return p, results
}

func TestBuildText_UnknownVersion_NoPatientSection(t *testing.T) {
	p, results := makeTestPrescriptionUnknown()
	text := output.BuildText(p, results)
	if strings.Contains(text, "--- 患者情報 ---") {
		t.Error("should not output patient section for unknown version")
	}
	if !strings.Contains(text, "=== VALIDATION RESULTS ===") {
		t.Error("expected validation section even for unknown version")
	}
}
```

`output_test.go` の先頭 import に `"strings"` を追加（すでにあれば不要）。

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test ./pkg/output/... -run "TestBuildText_Ver2_1|TestBuildText_Unknown" -v
```

Expected: `TestBuildText_Ver2_1_ShowsRecord1AsPatient` が FAIL（現在の `BuildText` はレコード2のみ参照）。

- [ ] **Step 3: `output_test.go` の古い定数を更新**

`makeTestPrescription`（24行）の `Version4` を `Version1_1` に変更（旧フォーマットのレコード2/6を使うテストデータのため）:

```go
	p := parser.Prescription{
		Version:   parser.Version1_1,
		RawQRs:    []string{"1,2,131012345\n2,山田太郎,ヤマダタロウ,19700101,1"},
		Records:   records,
		RecordMap: rm,
	}
```

- [ ] **Step 4: `output.go` の `BuildText` にバージョン分岐を追加**

`pkg/output/output.go` の `BuildText` 関数内、`=== PARSED DATA ===` ブロック（25〜70行）を以下に置き換える:

```go
	sb.WriteString("=== PARSED DATA ===\n")
	fmt.Fprintf(&sb, "バージョン: %s\n", p.Version)

	if p.Version == parser.VersionUnknown {
		sb.WriteString("\n")
	} else if p.Version >= parser.Version2_1 {
		// JAHISTC04以降: レコード1=患者情報, レコード201以上=薬品情報
		if recs, ok := p.RecordMap["1"]; ok && len(recs) > 0 {
			sb.WriteString("--- 患者情報 ---\n")
			f := recs[0].Fields
			if len(f) > 1 {
				fmt.Fprintf(&sb, "氏名: %s\n", f[1])
			}
			if len(f) > 3 {
				fmt.Fprintf(&sb, "生年月日: %s\n", formatDate(f[3]))
			}
			if len(f) > 2 {
				fmt.Fprintf(&sb, "性別: %s\n", formatSex(f[2]))
			}
		}
		if recs, ok := p.RecordMap["201"]; ok {
			sb.WriteString("--- Rp情報 ---\n")
			for i, rec := range recs {
				f := rec.Fields
				name, dose, unit := "", "", ""
				if len(f) > 2 {
					name = f[2]
				}
				if len(f) > 3 {
					dose = f[3]
				}
				if len(f) > 4 {
					unit = f[4]
				}
				fmt.Fprintf(&sb, "Rp%d: %s %s%s\n", i+1, name, dose, unit)
			}
		}
	} else {
		// JAHISTC01〜03: レコード2=患者情報, レコード6=薬品情報
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
				name, dose, unit, days := "", "", "", ""
				if len(f) > 2 {
					name = f[2]
				}
				if len(f) > 3 {
					dose = f[3]
				}
				if len(f) > 4 {
					unit = f[4]
				}
				if len(f) > 6 {
					days = f[6]
				}
				fmt.Fprintf(&sb, "Rp%d: %s %s%s %s日分\n", i+1, name, dose, unit, days)
			}
		}
	}

	sb.WriteString("\n")
```

- [ ] **Step 5: 全outputテストを実行**

```bash
go test ./pkg/output/... -v
```

Expected: 全テスト PASS。

- [ ] **Step 6: コミット**

```bash
git add pkg/output/output.go pkg/output/output_test.go
git commit -m "fix: BuildText にバージョン分岐を追加（JAHISTC04以降/以前/Unknown）"
```

---

### Task 7: `integration_test.go` — 定数更新

**Files:**
- Modify: `pkg/parser/integration_test.go`

- [ ] **Step 1: 古い定数参照を確認**

```bash
grep -n "Version2\|Version3\|Version4" pkg/parser/integration_test.go
```

Expected: `parser.Version4`, `parser.Version2`, `parser.Version3` の参照が複数行。

- [ ] **Step 2: 定数を更新**

`pkg/parser/integration_test.go` の `pipelineCases` スライス（19〜27行）を以下に置き換える:

```go
var pipelineCases = []struct {
	imageFile   string
	wantVersion parser.Version
	wantRecord  string
}{
	{"qr_ver4_single.png", parser.Version2_1, "1"},
	{"qr_ver2_single.png", parser.Version1_1, "2"},
	{"qr_ver3_single.png", parser.Version2_0, "2"},
}
```

`TestPipeline_SplitQR_Combined`（81行）を更新:

```go
	if p.Version != parser.Version2_1 {
		t.Errorf("version: want Version2_1, got %v", p.Version)
	}
```

- [ ] **Step 3: 統合テストを実行**

```bash
go test ./pkg/parser/... -run "TestPipeline" -v
```

Expected: 全テスト PASS。

- [ ] **Step 4: 全テストスイートを実行**

```bash
go test ./... -v 2>&1 | tail -30
```

Expected: 全パッケージで PASS。

- [ ] **Step 5: コミット**

```bash
git add pkg/parser/integration_test.go
git commit -m "fix: integration_test の定数を Version2_1/Version1_1/Version2_0 に更新"
```

---

### Task 8: ドキュメント更新

**Files:**
- Modify: `docs/superpowers/specs/2026-04-25-jahis-qr-cli-design.md`

- [ ] **Step 1: 既存バージョン記述を確認**

```bash
grep -n "Ver\.\|バージョン\|JAHISTC" docs/superpowers/specs/2026-04-25-jahis-qr-cli-design.md | head -30
```

- [ ] **Step 2: バージョン記述セクションを更新**

既存スペックのバージョン関連の記述（「対応バージョン」「バージョン検出」等）を以下の内容に更新する:

```markdown
## 対応バージョン

| JAHIS仕様書 | データフォーマット | JAHISCTヘッダー | バージョン定数 | レコード構造 |
|---|---|---|---|---|
| Ver.1.0 (2012/9) | 1 | JAHISTC01 | `Version1_0` | レコード2=患者, レコード6=薬品 |
| Ver.1.1 (2013/9) | 2 | JAHISTC02 | `Version1_1` | レコード2=患者, レコード6=薬品 |
| Ver.2.0 (2015/11) | 3 | JAHISTC03 | `Version2_0` | レコード2=患者, レコード6=薬品 |
| Ver.2.1 (2016/3) | 4 | JAHISTC04 | `Version2_1` | レコード1=患者, レコード201=薬品 |
| Ver.2.2 (2017/11) | 5 | JAHISTC05 | `Version2_2` | レコード1=患者, レコード201=薬品 |
| Ver.2.3 (2019/4) | 6 | JAHISTC06 | `Version2_3` | レコード1=患者, レコード201=薬品 |
| Ver.2.4 (2020/3) | 7 | JAHISTC07 | `Version2_4` | レコード1=患者, レコード201=薬品 |
| Ver.2.5 (2024/3) | 8 | JAHISTC08 | `Version2_5` | レコード1=患者, レコード201=薬品 |
| Ver.2.6 (2024/9) | 9 | JAHISTC09 | `Version2_6` | レコード1=患者, レコード201=薬品 |

**注意:** 旧バージョン（JAHISTC01〜03）は `JAHISTC{N}` の1桁形式でも記録されている場合がある。
バージョンを検出できない場合はエラーとして処理し、フォールバックは行わない。
```

- [ ] **Step 3: コミット**

```bash
git add docs/superpowers/specs/2026-04-25-jahis-qr-cli-design.md
git commit -m "docs: JAHIS対応バージョン表を Ver.1.0〜2.6 (JAHISTC01〜09) に更新"
```

---

## 完了確認

全タスク完了後:

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
