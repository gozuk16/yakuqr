# パーサー堅牢性テストデータ拡充 設計書

**日付:** 2026-04-29
**ブランチ:** feat/911-split-support（または新規ブランチ）

---

## 概要

JAHIS処方箋QRコードの分割ケースにおけるパーサーの堅牢性を検証するため、
3分割・途中欠落・部分スキャンの各シナリオをカバーするテストデータと統合テストを追加する。

---

## 追加するテストデータ（6ファイル）

### JAHISTC連番 3分割（JAHISTC04 = Ver.2.1）

**患者:** テスト九郎（男性, 1990-09-01）  
**処方:** Rp1〜Rp3の3種類

| ファイル | ヘッダー | 収録レコード |
|---|---|---|
| `testdata/ver4_split3_1.txt` | JAHISTC04,1 | 1（患者）, 5（処方日）, 11（薬局） |
| `testdata/ver4_split3_2.txt` | JAHISTC04,2 | 15（薬剤師）, 201/301×Rp1 |
| `testdata/ver4_split3_3.txt` | JAHISTC04,3 | 201/301×Rp2, 201/301×Rp3 |

薬品:
- Rp1: アムロジピン錠5mg「日医工」 1錠×28日分（1日1回 朝食後）
- Rp2: メトホルミン塩酸塩錠500mg「サワイ」 2錠×28日分（1日2回 朝夕食後）
- Rp3: ロスバスタチン錠5mg「トーワ」 1錠×28日分（1日1回 就寝前）

### 911 3分割（JAHISTC08 = Ver.2.6、累積型）

全QRがJAHISTC08,1（同一seq）、末尾に 911,00000000000002,3,N を持つ累積型。  
QRNは直前までの全レコードを包含する。

| ファイル | 911レコード | 収録レコード（累積） |
|---|---|---|
| `testdata/ver2_6_911split3_1.txt` | 911,...,3,1 | ヘッダー + Rp1 |
| `testdata/ver2_6_911split3_2.txt` | 911,...,3,2 | ヘッダー + Rp1 + Rp2 |
| `testdata/ver2_6_911split3_3.txt` | 911,...,3,3 | ヘッダー + Rp1 + Rp2 + Rp3（完全） |

---

## テストケース（6ケース）

既存の `qr_ver4_split_1.png` と `qr_ver2_6_911split_1.png` を再利用する。

### JAHISTC連番

| # | テスト名 | 入力QR | 期待結果 |
|---|---|---|---|
| 1 | 3分割・全枚 | split3_1+2+3 | Rp1〜Rp3全て取得, WARNINGなし |
| 2 | 3分割・QR2欠落 | split3_1+3 | `[WARNING] 2/3枚目が見つかりません`, Rp1なし, Rp2/3あり |
| 3 | 2分割・QR1のみ | split_1のみ | WARNINGなし, 薬品レコードなし（QR2に薬品がある） |

### 911分割

| # | テスト名 | 入力QR | 期待結果 |
|---|---|---|---|
| 4 | 3分割・全枚 | 911split3_1+2+3 | QR3の累積からRp1〜Rp3取得, 911レコードなし |
| 5 | 3分割・QR2欠落 | 911split3_1+3 | QR3が最大連番→Rp1〜Rp3全て取得（累積の効果）, 911レコードなし |
| 6 | 2分割・QR1のみ | 911split_1のみ | `[WARNING] 分割制御レコード（911）が検出されました...`, Rp1のみ |

---

## バリデーター追加（pkg/validator/qr_checks.go）

### checkSplit911Incomplete

`RecordMap["911"]` が存在する場合、911行がパーサーで除去されなかったことを示す。
これは全分割QRが揃わず911パスに乗れなかったケース（単独QRスキャン）に相当する。

```
[WARNING] 分割制御レコード（911）が検出されました。
          分割QRの一部が未取得の可能性があります
```

- Level: LevelWarning
- Field: "分割制御レコード 911"

`Validate()` に追加する呼び出し:
```go
results = append(results, checkSplit911Incomplete(p)...)
```

---

## 変更ファイル一覧

```
testdata/ver4_split3_1.txt              新規
testdata/ver4_split3_2.txt              新規
testdata/ver4_split3_3.txt              新規
testdata/ver2_6_911split3_1.txt         新規
testdata/ver2_6_911split3_2.txt         新規
testdata/ver2_6_911split3_3.txt         新規
testdata/generated/qr_ver4_split3_*.png 生成
testdata/generated/qr_ver2_6_911split3_*.png 生成
pkg/validator/qr_checks.go              checkSplit911Incomplete 追加
pkg/validator/validator.go              Validate() に呼び出し追加
pkg/parser/integration_test.go          6ケース追加
```

---

## スコープ外

- バリデーターの 911 WARNING に QR番号を含める（どのQRが欠けているか特定は複雑なため別タスク）
- JAHISTC連番 2分割・QR2欠落のケースに WARNING を追加（現在は WARNING なし。仕様上、何枚あるか分からないため意図的）
- Ver.2.2〜2.4 の単体テストデータ追加（別タスク）
