# JAHISパーサー バージョン検出修正 設計書

**日付:** 2026-04-27
**プロジェクト:** yakuqr
**ブランチ:** fix/jahis-parser-version

---

## 概要

JAHIS電子版お薬手帳データフォーマット仕様書（Ver.1.0〜Ver.2.6）の全バージョンに対応するため、Go パーサーのバージョン検出ロジックを修正する。ユーザーが `types.go` で定義した新しいバージョン定数（`Version1_0`〜`Version2_6`）に合わせ、関連するすべてのファイルを整合させる。

---

## バージョンマッピング（仕様書確認済み）

| JAHIS仕様書 | データフォーマット | JAHISCTヘッダー | Go定数 | 定数値 |
|---|---|---|---|---|
| Ver.1.0 (2012/9) | 1 | JAHISTC01 | `Version1_0` | 1 |
| Ver.1.1 (2013/9) | 2 | JAHISTC02 | `Version1_1` | 2 |
| Ver.2.0 (2015/11) | 3 | JAHISTC03 | `Version2_0` | 3 |
| Ver.2.1 (2016/3) | 4 | JAHISTC04 | `Version2_1` | 4 |
| Ver.2.2 (2017/11) | 5 | JAHISTC05 | `Version2_2` | 5 |
| Ver.2.3 (2019/4) | 6 | JAHISTC06 | `Version2_3` | 6 |
| Ver.2.4 (2020/3) | 7 | JAHISTC07 | `Version2_4` | 7 |
| Ver.2.5 (2024/3) | 8 | JAHISTC08 | `Version2_5` | 8 |
| Ver.2.6 (2024/9) | 9 | JAHISTC09 | `Version2_6` | 9 |

**重要:** JAHISTC番号と `Version` 定数値が一致する（JAHISTC`XX` → `Version(XX)`）。

---

## ヘッダー形式

仕様書に基づくヘッダー行の形式：

```
JAHISTC{vv},{seq}
```

- `{vv}` は2桁ゼロ埋め（新バージョン固定）。旧実装では1桁（`JAHISTC1`）も存在し得る。
- `{seq}` は分割QRの通し番号（1始まり）。

バージョン検出時は `"0"` をトリムして数値比較するため、1桁・2桁の両方を自動吸収する。

---

## レコード構造の分岐

| バージョン | 患者情報 | 薬品情報 | 処方箋情報 |
|---|---|---|---|
| JAHISTC01〜03（Ver.1.0〜2.0） | レコード2 | レコード6 | レコード1（フィールド[1]=バージョン番号） |
| JAHISTC04〜09（Ver.2.1〜2.6） | レコード1 | レコード201以上 | （ヘッダーに統合） |

コード内では `v >= Version2_1` で分岐する。

---

## 修正方針

### 1. `pkg/parser/parser.go` — `detectVersion`

**現状の問題:**
- JAHISTC02/03/04 のみ対応（01・05〜09 が未対応）
- 古い定数名（`Version2/3/4`）を参照（コンパイルエラー）
- 1桁形式（`JAHISTC1`）を `len(first) >= 9` ガードで弾いている
- バージョン未検出時に `Version4` へフォールバック（不適切）

**修正後の動作:**
```
1. JAHISCTヘッダーがあれば数値を解析 → Version(n) を返す
2. ヘッダーなしなら旧フォーマットと判断 → レコード1のフィールド[1]を参照
3. 検出できなければ VersionUnknown + "[ERROR]" メッセージ
```

実装:
```go
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
    // ヘッダーなし旧フォーマット: レコード1のフィールド[1]がバージョン番号
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

### 2. `pkg/parser/types.go` — `Version.String()`

現在 `Version1_0`/`Version1_1`/`Version2_0`/`VersionUnknown` が欠落。追加する：

```go
case VersionUnknown:
    return "Unknown"
case Version1_0:
    return "Ver.1.0"
case Version1_1:
    return "Ver.1.1"
case Version2_0:
    return "Ver.2.0"
```

### 3. `pkg/validator/rules.go` — `rulesFor`

```go
func rulesFor(v parser.Version) []rule {
    if v == parser.VersionUnknown {
        return unknownVersionRules()
    }
    if v >= parser.Version2_1 {
        return ver4Rules()  // レコード1=患者, レコード201=薬品
    }
    return ver2ver3Rules(v)  // レコード2=患者, レコード6=薬品
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

バージョン互換性INFOメッセージの対象も `ver2ver3Rules` 内で `v <= Version2_0` に限定する。

### 4. `pkg/output/output.go` — `BuildText`

`v >= parser.Version2_1` でレコード参照先を分岐：

- `Version2_1` 以上: レコード1から患者情報（氏名=f[1], 生年月日=f[3]）
- `Version2_0` 以下: レコード2から患者情報（現在と同じ）
- `VersionUnknown`: 患者情報・薬品情報セクションをスキップ

### 5. テストデータの追加

| ファイル | 内容 |
|---|---|
| `testdata/ver1_0_single.txt` | JAHISTC01形式（2桁）の最小サンプル |
| `testdata/ver1_0_1digit.txt` | JAHISTC1形式（1桁）の最小サンプル |
| `testdata/ver2_5_single.txt` | JAHISTC08形式の最小サンプル |
| `testdata/ver2_6_single.txt` | JAHISTC09形式の最小サンプル |

JAHISTC01/02/03 は旧フォーマット（レコード2=患者, レコード6=薬品）と同じ構造。

### 6. テストの更新

`parser_test.go` / `integration_test.go` / `validator_test.go` 内の古い定数を置換：

| 旧 | 新 |
|---|---|
| `parser.Version2` | `parser.Version1_1` |
| `parser.Version3` | `parser.Version2_0` |
| `parser.Version4` | `parser.Version2_1` |

加えて以下のテストケースを追加：
- `TestParse_JAHISTC01_DetectsVersion` （JAHISTC01形式）
- `TestParse_JAHISTC1_1digit_DetectsVersion` （1桁形式）
- `TestParse_JAHISTC08_DetectsVersion` （Ver.2.5）
- `TestParse_JAHISTC09_DetectsVersion` （Ver.2.6）
- `TestParse_VersionUnknown_ReturnsError` （未検出時ERRORメッセージ確認）
- `TestValidate_UnknownVersion_ReturnsError` （バリデーターのエラー確認）

### 7. ドキュメント更新

`docs/superpowers/specs/2026-04-25-jahis-qr-cli-design.md` のバージョン関連記述を本設計書の内容に合わせて更新。

---

## 修正対象ファイル一覧

```
pkg/parser/parser.go          ← detectVersion 全面修正
pkg/parser/types.go           ← Version.String() 補完
pkg/validator/rules.go        ← rulesFor に VersionUnknown 分岐追加
pkg/output/output.go          ← BuildText にバージョン分岐追加
pkg/parser/parser_test.go     ← 定数更新 + テスト追加
pkg/parser/integration_test.go← 定数更新
pkg/validator/validator_test.go← 定数更新 + VersionUnknown テスト追加
testdata/ver1_0_single.txt    ← 新規追加
testdata/ver1_0_1digit.txt    ← 新規追加
testdata/ver2_5_single.txt    ← 新規追加
testdata/ver2_6_single.txt    ← 新規追加
docs/superpowers/specs/2026-04-25-jahis-qr-cli-design.md ← バージョン表更新
```

---

## 非対応事項（スコープ外）

- Swift（iOS）側の修正（別タスクとして後日実施）
- JAHISTC10以降の将来バージョン対応
- レコード構造の詳細バリデーション（Ver.1.0〜2.0の各フィールド仕様差異）
