# パーサー堅牢性テストデータ拡充 実装計画

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** JAHIS処方箋QRコードの3分割・途中欠落・部分スキャンの各シナリオに対してテストデータ・バリデーター・統合テストを追加し、パーサーの堅牢性を検証する。

**Architecture:** テストデータ（txtファイル+PNG）を追加し、gen-testdata-qrで画像生成する。バリデーターに911残存検出チェックをTDDで追加する。最後に統合テスト6ケースを追加して全シナリオを網羅する。

**Tech Stack:** Go 1.21+, github.com/skip2/go-qrcode（QR生成）, github.com/makiuchi-d/gozxing（QRデコード）

---

## ファイル構成

```
testdata/ver4_split3_1.txt              新規: JAHISTC04,1（ヘッダー+患者情報）
testdata/ver4_split3_2.txt              新規: JAHISTC04,2（薬剤師+Rp1）
testdata/ver4_split3_3.txt              新規: JAHISTC04,3（Rp2+Rp3）
testdata/ver2_6_911split3_1.txt         新規: JAHISTC08,1 累積Rp1 + 911,ID,3,1
testdata/ver2_6_911split3_2.txt         新規: JAHISTC08,1 累積Rp1+Rp2 + 911,ID,3,2
testdata/ver2_6_911split3_3.txt         新規: JAHISTC08,1 累積Rp1+Rp2+Rp3 + 911,ID,3,3
testdata/generated/qr_ver4_split3_*.png 生成: go run ./tools/gen-testdata-qr/. で作成
testdata/generated/qr_ver2_6_911split3_*.png 生成: 同上
pkg/validator/qr_checks.go             変更: checkSplit911Incomplete 追加
pkg/validator/validator.go              変更: Validate() に呼び出し追加
pkg/validator/validator_test.go         変更: TestValidate_911RecordPresent_ReturnsWarning 追加
pkg/parser/integration_test.go          変更: 6ケース追加
```

---

### Task 1: テストデータファイル作成と QR 画像生成

**Files:**
- Create: `testdata/ver4_split3_1.txt`
- Create: `testdata/ver4_split3_2.txt`
- Create: `testdata/ver4_split3_3.txt`
- Create: `testdata/ver2_6_911split3_1.txt`
- Create: `testdata/ver2_6_911split3_2.txt`
- Create: `testdata/ver2_6_911split3_3.txt`

- [ ] **Step 1: JAHISTC連番3分割ファイルを作成する**

`testdata/ver4_split3_1.txt` を作成:
```
JAHISTC04,1
1,テスト九郎,1,19900901,100-0001,東京都千代田区千代田1-1,,,0,テスト
5,20240601,1
11,薬局テスト,13,4,1234567,1000001,東京都千代田区1-1,03-1234-5678,1
```

`testdata/ver4_split3_2.txt` を作成:
```
JAHISTC04,2
15,医師テスト,,1
201,1,アムロジピン錠5mg「日医工」,1,錠,4,1149019F1625,1,,,
301,1,（1日1回　朝食後服用）,28,日分,1,1,,1
```

`testdata/ver4_split3_3.txt` を作成:
```
JAHISTC04,3
201,2,メトホルミン塩酸塩錠500mg「サワイ」,2,錠,4,3969018F1122,1,,,
301,2,（1日2回　朝夕食後服用）,28,日分,1,1,,1
201,3,ロスバスタチン錠5mg「トーワ」,1,錠,4,2189019F1026,1,,,
301,3,（1日1回　就寝前服用）,28,日分,1,1,,1
```

- [ ] **Step 2: 911分割3分割ファイルを作成する**

`testdata/ver2_6_911split3_1.txt` を作成（Rp1のみ）:
```
JAHISTC08,1
1,テスト九郎,1,19900901,100-0001,東京都千代田区千代田1-1,,,0,テスト
5,20240601,1
11,薬局テスト,13,4,1234567,1000001,東京都千代田区1-1,03-1234-5678,1
15,医師テスト,,1
201,1,アムロジピン錠5mg「日医工」,1,錠,4,1149019F1625,1,,,
301,1,（1日1回　朝食後服用）,28,日分,1,1,,1
911,00000000000002,3,1
```

`testdata/ver2_6_911split3_2.txt` を作成（Rp1+Rp2 累積）:
```
JAHISTC08,1
1,テスト九郎,1,19900901,100-0001,東京都千代田区千代田1-1,,,0,テスト
5,20240601,1
11,薬局テスト,13,4,1234567,1000001,東京都千代田区1-1,03-1234-5678,1
15,医師テスト,,1
201,1,アムロジピン錠5mg「日医工」,1,錠,4,1149019F1625,1,,,
301,1,（1日1回　朝食後服用）,28,日分,1,1,,1
201,2,メトホルミン塩酸塩錠500mg「サワイ」,2,錠,4,3969018F1122,1,,,
301,2,（1日2回　朝夕食後服用）,28,日分,1,1,,1
911,00000000000002,3,2
```

`testdata/ver2_6_911split3_3.txt` を作成（Rp1+Rp2+Rp3 累積・完全）:
```
JAHISTC08,1
1,テスト九郎,1,19900901,100-0001,東京都千代田区千代田1-1,,,0,テスト
5,20240601,1
11,薬局テスト,13,4,1234567,1000001,東京都千代田区1-1,03-1234-5678,1
15,医師テスト,,1
201,1,アムロジピン錠5mg「日医工」,1,錠,4,1149019F1625,1,,,
301,1,（1日1回　朝食後服用）,28,日分,1,1,,1
201,2,メトホルミン塩酸塩錠500mg「サワイ」,2,錠,4,3969018F1122,1,,,
301,2,（1日2回　朝夕食後服用）,28,日分,1,1,,1
201,3,ロスバスタチン錠5mg「トーワ」,1,錠,4,2189019F1026,1,,,
301,3,（1日1回　就寝前服用）,28,日分,1,1,,1
911,00000000000002,3,3
```

- [ ] **Step 3: QR画像を生成する**

```bash
go run ./tools/gen-testdata-qr/.
```

期待出力（関連部分）:
```
generated: .../testdata/generated/qr_ver4_split3_1.png
generated: .../testdata/generated/qr_ver4_split3_2.png
generated: .../testdata/generated/qr_ver4_split3_3.png
generated: .../testdata/generated/qr_ver2_6_911split3_1.png
generated: .../testdata/generated/qr_ver2_6_911split3_2.png
generated: .../testdata/generated/qr_ver2_6_911split3_3.png
```

- [ ] **Step 4: ビルドとテストが通ることを確認する**

```bash
go build ./... && go test ./...
```

期待: `ok` が全パッケージで出ること。

- [ ] **Step 5: コミットする**

```bash
git add testdata/ver4_split3_1.txt testdata/ver4_split3_2.txt testdata/ver4_split3_3.txt \
        testdata/ver2_6_911split3_1.txt testdata/ver2_6_911split3_2.txt testdata/ver2_6_911split3_3.txt \
        testdata/generated/qr_ver4_split3_1.png testdata/generated/qr_ver4_split3_2.png \
        testdata/generated/qr_ver4_split3_3.png \
        testdata/generated/qr_ver2_6_911split3_1.png testdata/generated/qr_ver2_6_911split3_2.png \
        testdata/generated/qr_ver2_6_911split3_3.png
git commit -m "feat: JAHISTC3分割・911分割3分割のテストデータを追加"
```

---

### Task 2: バリデーター — checkSplit911Incomplete 追加（TDD）

911レコードがパースされたデータに残っている場合（1枚だけスキャンした等）に WARNING を出す。

**Files:**
- Modify: `pkg/validator/qr_checks.go`
- Modify: `pkg/validator/validator.go`
- Modify: `pkg/validator/validator_test.go`

- [ ] **Step 1: 失敗するユニットテストを書く**

`pkg/validator/validator_test.go` の末尾に追加:

```go
func TestValidate_911RecordPresent_ReturnsWarning(t *testing.T) {
	// 911レコードがRecordMapに残っている処方箋（QR1だけスキャンした状態）
	p := parser.Prescription{
		Version: parser.Version2_6,
		RecordMap: map[string][]parser.Record{
			"1": {{Type: "1", Fields: []string{"1", "テスト九郎", "1", "19900901"}}},
			"201": {{Type: "201", Fields: []string{"201", "1", "アムロジピン錠5mg", "1", "錠", "4", "1149019F1625", "1"}}},
			"911": {{Type: "911", Fields: []string{"911", "00000000000002", "3", "1"}}},
		},
	}
	results := validator.Validate(p)
	found := false
	for _, r := range results {
		if r.Level == validator.LevelWarning && r.Field == "分割制御レコード 911" {
			found = true
		}
	}
	if !found {
		t.Error("want WARNING for 911 record presence, got none")
	}
}
```

- [ ] **Step 2: テストが失敗することを確認する**

```bash
go test ./pkg/validator/... -run TestValidate_911RecordPresent_ReturnsWarning -v
```

期待: `FAIL — want WARNING for 911 record presence, got none`

- [ ] **Step 3: checkSplit911Incomplete を実装する**

`pkg/validator/qr_checks.go` の `checkOrphanRecords` 関数の後に追加:

```go
// checkSplit911Incomplete は RecordMap に 911 レコードが残存する場合 WARNING を返す。
// 911 レコードは通常 parse911Parts によって除去されるが、
// 分割 QR が 1 枚しかスキャンできなかった場合は除去されずに残る。
func checkSplit911Incomplete(p parser.Prescription) []ValidationResult {
	if _, ok := p.RecordMap["911"]; !ok {
		return nil
	}
	return []ValidationResult{{
		Level:   LevelWarning,
		Field:   "分割制御レコード 911",
		Message: "分割制御レコード（911）が検出されました。分割QRの一部が未取得の可能性があります",
	}}
}
```

- [ ] **Step 4: Validate() に呼び出しを追加する**

`pkg/validator/validator.go` を以下のように変更する:

```go
func Validate(p parser.Prescription) []ValidationResult {
	// QR読み取り品質の共通チェック（全バージョン）
	results := checkQRReadFailures(p)
	results = append(results, checkGarbledData(p)...)
	results = append(results, checkOrphanRecords(p)...)
	results = append(results, checkSplit911Incomplete(p)...)

	// バージョン固有チェック
	for _, r := range rulesFor(p.Version) {
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

- [ ] **Step 5: テストが通ることを確認する**

```bash
go test ./pkg/validator/... -v
```

期待: `PASS` for `TestValidate_911RecordPresent_ReturnsWarning` および既存テスト全て。

- [ ] **Step 6: コミットする**

```bash
git add pkg/validator/qr_checks.go pkg/validator/validator.go pkg/validator/validator_test.go
git commit -m "feat: 911分割制御レコード残存をバリデーターのWARNINGで検出"
```

---

### Task 3: 統合テスト 6 ケース追加

**Files:**
- Modify: `pkg/parser/integration_test.go`

- [ ] **Step 1: JAHISTC 3分割テスト（全枚・QR2欠落・QR1のみ）を書く**

`pkg/parser/integration_test.go` の末尾に追加:

```go
func TestPipeline_Split3Way_All(t *testing.T) {
	qrs1, _ := decoder.DecodeFile(generatedImagePath("qr_ver4_split3_1.png"))
	qrs2, _ := decoder.DecodeFile(generatedImagePath("qr_ver4_split3_2.png"))
	qrs3, _ := decoder.DecodeFile(generatedImagePath("qr_ver4_split3_3.png"))

	allQRs := toRawQRs(append(append(qrs1, qrs2...), qrs3...))
	p, msgs := parser.Parse(allQRs)
	for _, m := range msgs {
		t.Logf("parse message: %s", m)
	}

	if p.Version != parser.Version2_1 {
		t.Errorf("version: want Version2_1, got %v", p.Version)
	}
	// QR2にRp1、QR3にRp2+Rp3が入っているので3件揃うこと
	if len(p.RecordMap["201"]) != 3 {
		t.Errorf("201レコード数: want 3, got %d", len(p.RecordMap["201"]))
	}
	for _, m := range msgs {
		if strings.Contains(m, "WARNING") {
			t.Errorf("unexpected parse WARNING: %s", m)
		}
	}
}

func TestPipeline_Split3Way_MissingQR2(t *testing.T) {
	qrs1, _ := decoder.DecodeFile(generatedImagePath("qr_ver4_split3_1.png"))
	qrs3, _ := decoder.DecodeFile(generatedImagePath("qr_ver4_split3_3.png"))

	allQRs := toRawQRs(append(qrs1, qrs3...))
	p, msgs := parser.Parse(allQRs)

	// パーサーが "2/3枚目が見つかりません" を警告すること
	found := false
	for _, m := range msgs {
		if strings.Contains(m, "2/3") {
			found = true
		}
	}
	if !found {
		t.Errorf("want parse warning about missing 2/3, got: %v", msgs)
	}

	// QR2(Rp1)が欠落しているので 201 レコードは Rp2,Rp3 の2件のみ
	rp201 := p.RecordMap["201"]
	if len(rp201) != 2 {
		t.Errorf("201レコード数: want 2 (Rp2+Rp3), got %d", len(rp201))
	}
}

func TestPipeline_Split2Way_QR1Only(t *testing.T) {
	// 既存の ver4_split_1.png を再利用（QR2は読まない）
	qrs1, errs1 := decoder.DecodeFile(generatedImagePath("qr_ver4_split_1.png"))
	if len(errs1) > 0 {
		t.Fatalf("decode errors: %v", errs1)
	}

	allQRs := toRawQRs(qrs1)
	p, _ := parser.Parse(allQRs)

	if p.Version != parser.Version2_1 {
		t.Errorf("version: want Version2_1, got %v", p.Version)
	}
	// QR1には薬品レコードがないこと（薬品はQR2にある）
	if _, ok := p.RecordMap["201"]; ok {
		t.Error("want no 201 records when only QR1 is scanned")
	}
}
```

- [ ] **Step 2: 911 3分割テスト（全枚・QR2欠落・QR1のみ）を書く**

同じファイルの末尾に続けて追加:

```go
func TestPipeline_911Split3Way_All(t *testing.T) {
	qrs1, _ := decoder.DecodeFile(generatedImagePath("qr_ver2_6_911split3_1.png"))
	qrs2, _ := decoder.DecodeFile(generatedImagePath("qr_ver2_6_911split3_2.png"))
	qrs3, _ := decoder.DecodeFile(generatedImagePath("qr_ver2_6_911split3_3.png"))

	allQRs := toRawQRs(append(append(qrs1, qrs2...), qrs3...))
	p, msgs := parser.Parse(allQRs)
	for _, m := range msgs {
		t.Logf("parse message: %s", m)
	}

	if p.Version != parser.Version2_6 {
		t.Errorf("version: want Version2_6, got %v", p.Version)
	}
	// QR3（最大連番、累積）からRp1〜Rp3の3件が取得できること
	if len(p.RecordMap["201"]) != 3 {
		t.Errorf("201レコード数: want 3, got %d", len(p.RecordMap["201"]))
	}
	// 911行は除去されていること
	if _, ok := p.RecordMap["911"]; ok {
		t.Error("911レコードがRecordMapに残っている")
	}

	results := validator.Validate(p)
	for _, r := range results {
		if r.Level == validator.LevelError {
			t.Errorf("unexpected validation ERROR: %v", r)
		}
	}
}

func TestPipeline_911Split3Way_MissingQR2(t *testing.T) {
	// QR2を省いてQR1+QR3のみ渡す
	qrs1, _ := decoder.DecodeFile(generatedImagePath("qr_ver2_6_911split3_1.png"))
	qrs3, _ := decoder.DecodeFile(generatedImagePath("qr_ver2_6_911split3_3.png"))

	allQRs := toRawQRs(append(qrs1, qrs3...))
	p, _ := parser.Parse(allQRs)

	if p.Version != parser.Version2_6 {
		t.Errorf("version: want Version2_6, got %v", p.Version)
	}
	// 累積型: QR3が最大連番なのでQR2が欠落してもRp1〜Rp3全て取得できる
	if len(p.RecordMap["201"]) != 3 {
		t.Errorf("201レコード数: want 3 (cumulative from QR3), got %d", len(p.RecordMap["201"]))
	}
	// 911行は除去されていること
	if _, ok := p.RecordMap["911"]; ok {
		t.Error("911レコードがRecordMapに残っている")
	}

	results := validator.Validate(p)
	for _, r := range results {
		if r.Level == validator.LevelError {
			t.Errorf("unexpected validation ERROR: %v", r)
		}
	}
}

func TestPipeline_911Split2Way_QR1Only(t *testing.T) {
	// 既存の ver2_6_911split_1.png を再利用（QR1のみスキャン）
	qrs1, errs1 := decoder.DecodeFile(generatedImagePath("qr_ver2_6_911split_1.png"))
	if len(errs1) > 0 {
		t.Fatalf("decode errors: %v", errs1)
	}

	allQRs := toRawQRs(qrs1)
	p, _ := parser.Parse(allQRs)

	// QR1のみなので allSameSeq=false → JAHISTC通常パスに落ちる
	// 911行がRecordMapに残るためバリデーターがWARNINGを出す
	results := validator.Validate(p)
	found := false
	for _, r := range results {
		if r.Level == validator.LevelWarning && r.Field == "分割制御レコード 911" {
			found = true
		}
	}
	if !found {
		t.Error("want WARNING about 911 record when only QR1 is scanned")
	}

	// QR1にはRp1が含まれているので201レコードが存在すること
	if len(p.RecordMap["201"]) == 0 {
		t.Error("want at least 1 Rp (Rp1) when QR1 is scanned")
	}
}
```

- [ ] **Step 3: `strings` パッケージのインポートが追加されているか確認・修正する**

`pkg/parser/integration_test.go` の import ブロックに `"strings"` が含まれていない場合は追加:

```go
import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gozuk16/yakuqr/pkg/decoder"
	"github.com/gozuk16/yakuqr/pkg/parser"
	"github.com/gozuk16/yakuqr/pkg/validator"
)
```

- [ ] **Step 4: テストを実行して全て通ることを確認する**

```bash
go test ./... -v 2>&1 | tail -20
```

期待: 全パッケージ `ok`, 新規テスト6ケースを含め全て `PASS`

- [ ] **Step 5: コミットする**

```bash
git add pkg/parser/integration_test.go
git commit -m "test: 3分割・途中欠落・部分スキャンの統合テスト6ケースを追加"
```

- [ ] **Step 6: ブランチをプッシュする**

```bash
git push origin feat/911-split-support
```
