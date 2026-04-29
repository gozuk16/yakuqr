# Swift/Go 完全同期 設計書

**日付:** 2026-04-29  
**ブランチ:** feat/swift-go-sync（新規）

---

## 概要

Go 実装（`pkg/parser`, `pkg/validator`）に加えられた以下の変更を、iOS Swift 実装（`ios/yakuqr-ios/`）に完全同期する。

1. 911 累積型分割パーサーのサポート
2. JAHISTC08（Ver.2.6）バージョン検出
3. バリデーター 4 チェック追加（`checkQRReadFailures`, `checkGarbledData`, `checkOrphanRecords`, `checkSplit911Incomplete`）
4. `RawQR` 型の導入と `JAHISPrescription.rawQRs` の型変更

---

## 変更ファイル一覧

```
ios/yakuqr-ios/yakuqr-ios/JAHIS/JAHISModels.swift          変更
ios/yakuqr-ios/yakuqr-ios/JAHIS/JAHISParser.swift          変更
ios/yakuqr-ios/yakuqr-ios/JAHIS/JAHISValidator.swift       変更
ios/yakuqr-ios/yakuqr-ios/Features/Scanner/ScannerViewModel.swift  変更
ios/yakuqr-ios/yakuqr-iosTests/JAHISTests/JAHISParserTests.swift   変更
ios/yakuqr-ios/yakuqr-iosTests/JAHISTests/JAHISValidatorTests.swift 変更
```

---

## セクション1: データモデル（JAHISModels.swift）

### RawQR 追加

```swift
struct RawQR {
    let text: String    // デコード成功時のテキスト（失敗時は ""）
    let errMsg: String  // 読み取り失敗メッセージ（成功時は ""）
    var isSuccess: Bool { errMsg.isEmpty }
}
```

Go の `parser.RawQR` と 1:1 対応。iOS では Vision Framework が QR 検出に成功した文字列のみ `addQR()` に渡すため、`errMsg` は常に空となるが、構造を Go と統一する。

### JAHISVersion リネームと拡張

| 旧 Swift | 新 Swift | Go 定数 | JAHISTC | 表示名 |
|---|---|---|---|---|
| `.v2` | `.v1_1` | `Version1_1` | JAHISTC02 / 旧フォーマット | Ver.1.1 |
| `.v3` | `.v2_0` | `Version2_0` | JAHISTC03 / 旧フォーマット | Ver.2.0 |
| `.v4` | `.v2_1` | `Version2_1` | JAHISTC04 | Ver.2.1 |
| ―（新規）| `.v2_6` | `Version2_6` | JAHISTC08 | Ver.2.6 |
| `.unknown` | `.unknown` | `VersionUnknown` | ― | Unknown |

`displayName` は Go の `Version.String()` 出力（`"Ver.2.1"` 等）に合わせる。

### JAHISPrescription 変更

```swift
struct JAHISPrescription {
    let version: JAHISVersion
    let rawQRs: [RawQR]      // [String] → [RawQR] に変更
    let records: [JAHISRecord]
    let recordMap: [String: [JAHISRecord]]
    let splitInfos: [JAHISSplitInfo]
}
```

---

## セクション2: パーサー（JAHISParser.swift）

### parse() シグネチャ変更

```swift
static func parse(_ rawQRs: [RawQR]) -> (JAHISPrescription, [String])
```

### combineQRs の変更

```swift
private static func combineQRs(_ rawQRs: [RawQR]) -> (String, [JAHISSplitInfo], [String]) {
    let texts = rawQRs.filter { $0.isSuccess }.map { $0.text }
    // texts を使って既存ロジック + 911 分割ロジックを実行
}
```

### 911 累積型分割ロジック追加

Go の `allSameSeq` / `parse911Parts` / `remove911Lines` を Swift に移植する。

```swift
// 全パーツが同一 seq を持つか判定（len >= 2 かつ全同一のとき true）
private static func allSameSeq(_ parts: [QRPart]) -> Bool

// 各パーツから 911 レコードを探す。全パーツに 911 がある場合のみ返す
private struct Part911 { let content: String; let total: Int; let current: Int }
private static func parse911Parts(_ parts: [QRPart]) -> [Part911]?

// 文字列から 911 行を除去
private static func remove911Lines(_ content: String) -> String
```

`combineQRs` 内の分岐:
1. `allSameSeq && parts.count > 1` → `parse911Parts` を試みる
2. 全パーツに 911 があれば: `current` 最大のパーツを採用し `remove911Lines` を適用
3. それ以外: 既存の JAHISTC 連番分割ロジック

### バージョン検出の拡張

```swift
// JAHISTC ヘッダーの番号を Int に変換して分岐
let numStr = String(first.dropFirst(7).split(separator: ",").first ?? "")
    .drop(while: { $0 == "0" })  // 先頭ゼロを除去
switch Int(numStr) {
case 1: return (.v1_0, nil)  // 将来対応用（旧フォーマットにフォールバック可）
case 2: return (.v1_1, nil)
case 3: return (.v2_0, nil)
case 4: return (.v2_1, nil)
case 8: return (.v2_6, nil)
default: break
}
// 旧フォーマット: record "1" の fields[1] で判定
switch fields[1] {
case "1": return (.v1_0, nil)
case "2": return (.v1_1, nil)
case "3": return (.v2_0, nil)
default: break
}
return (.v2_1, "[INFO] バージョンを検出できなかったため、Ver.2.1（最新版）として処理します")
```

---

## セクション3: バリデーター（JAHISValidator.swift）

### validate() の変更

```swift
static func validate(_ prescription: JAHISPrescription) -> [JAHISValidationResult] {
    var results: [JAHISValidationResult] = []
    results += checkQRReadFailures(prescription)
    results += checkGarbledData(prescription)
    results += checkOrphanRecords(prescription)
    results += checkSplit911Incomplete(prescription)
    results += rulesFor(prescription.version).compactMap { rule -> JAHISValidationResult? in
        let (ok, msg) = rule.check(prescription)
        guard !ok else { return nil }
        return JAHISValidationResult(level: rule.level, field: rule.field, message: msg)
    }
    return results
}
```

### 4チェックの仕様

**checkQRReadFailures**
- `rawQRs` の各要素で `errMsg` が非空なら WARNING
- Field: `"QR #N"`, Message: `"読み取り失敗（{errMsg}）。このQRに含まれるデータが欠落しています"`

**checkGarbledData**
- `records` の各フィールドに `"\u{FFFD}"` が含まれる場合 WARNING
- `findQRForRecord` で QR 番号を特定: 完全一致 → 前方一致フォールバック
- 同一レコード種別・同一 QR 番号の組み合わせは 1 件のみ報告
- Field: `"QR #N, レコード種別 T"` または `"レコード種別 T"`

**checkOrphanRecords**
- `records` の `type` が数字以外（空文字含む）なら WARNING
- Field: `"レコード種別"`, Message: `"不正なレコード種別 \"{type}\" が検出されました（QRコード欠落による断片データの可能性）"`

**checkSplit911Incomplete**
- `recordMap["911"]` が存在する場合 WARNING
- Field: `"分割制御レコード 911"`, Message: `"分割制御レコード（911）が検出されました。分割QRの一部が未取得の可能性があります"`

### バージョン分岐の整理

```swift
private static func rulesFor(_ version: JAHISVersion) -> [Rule] {
    switch version {
    case .v2_1, .v2_6, .unknown:
        return newFormatRules()   // 旧 ver4Rules(): レコード1, 201
    case .v1_0, .v1_1, .v2_0:
        return oldFormatRules(version)  // 旧 ver2ver3Rules(): レコード2, 6
    }
}
```

---

## セクション4: ScannerViewModel.swift

```swift
@Published var scannedQRs: [RawQR] = []
private var seenQRs: Set<String> = []   // 重複排除は String のまま

func addQR(_ value: String) {
    guard !seenQRs.contains(value) else { return }
    seenQRs.insert(value)
    scannedQRs.append(RawQR(text: value, errMsg: ""))
}

func parse() {
    let (prescription, msgs) = JAHISParser.parse(scannedQRs)
    let validations = JAHISValidator.validate(prescription)
    let text = PrescriptionTextFormatter.format(prescription, validations: validations)
    parseResult = ParseResult(
        prescription: prescription,
        validations: validations,
        messages: msgs,
        formattedText: text
    )
}
```

---

## セクション5: テスト

### JAHISParserTests 追加ケース

- `testParse_911split_3way_all_hasAllRps` — 3 分割全枚: version=.v2_6, 201×3件, 911なし
- `testParse_911split_missingQR2_usesMaxCumulative` — QR1+QR3: 累積型のため 201×3件
- `testParse_911split_qr1Only_has911InRecordMap` — QR1のみ: recordMap["911"] が存在
- `testParse_ver2_6_detectsVersion` — JAHISTC08ヘッダー → .v2_6

### JAHISParserTests 既存ケース更新

`JAHISParser.parse([raw])` → `JAHISParser.parse([RawQR(text: raw, errMsg: "")])`

### JAHISValidatorTests 追加ケース

- `testValidate_911RecordPresent_returnsWarning` — 911残存 → WARNING
- `testValidate_orphanRecord_returnsWarning` — 非数字レコード種別 → WARNING

### JAHISValidatorTests 既存ケース更新

`JAHISPrescription(rawQRs: ["dummy"], ...)` → `rawQRs: [RawQR(text: "dummy", errMsg: "")]`

---

## スコープ外

- `PrescriptionTextFormatter.swift` のバージョン表示文字列更新（`.v2_1` → `"Ver.2.1"` 等は `displayName` 経由なので自動対応）
- `ScannerViewModelTests.swift` の `scannedQRs` 型変更に伴う軽微な更新
- Ver.1.0 専用バリデーションルールの追加（現行 Go も旧フォーマットとして共通ルール使用のため対象外）
