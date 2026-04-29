# Swift/Go 完全同期 実装計画

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Go 実装に追加された 911 累積型分割パーサー・バージョン検出拡張・バリデーター 4 チェックを Swift iOS 実装に完全同期する。

**Architecture:** Task 1 でデータモデルの型変更（`RawQR` 追加・`JAHISVersion` リネーム）と全ファイルのコンパイル修正を行い、Task 2 でパーサーの 911 分割ロジックを TDD で追加し、Task 3 でバリデーターの 4 チェックを TDD で追加する。型変更は Swift の全ファイルに波及するため Task 1 は一括変更とする。

**Tech Stack:** Swift 5.9+, Xcode, XCTest, iOS Simulator (iPhone 17 Pro)

---

## ファイル構成

```
ios/yakuqr-ios/yakuqr-ios/JAHIS/JAHISModels.swift           変更: RawQR追加, JAHISVersion リネーム+拡張, JAHISPrescription.rawQRs型変更
ios/yakuqr-ios/yakuqr-ios/JAHIS/JAHISParser.swift           変更: [RawQR]引数, 911分割, バージョン検出拡張
ios/yakuqr-ios/yakuqr-ios/JAHIS/JAHISValidator.swift        変更: 4チェック追加, バージョン分岐整理
ios/yakuqr-ios/yakuqr-ios/JAHIS/PrescriptionTextFormatter.swift  変更: rawQRs反復, バージョン判定
ios/yakuqr-ios/yakuqr-ios/Features/Scanner/ScannerViewModel.swift  変更: scannedQRs型変更, addQR
ios/yakuqr-ios/yakuqr-iosTests/JAHISTests/JAHISParserTests.swift    変更: 引数型更新, バージョン名更新, 新テスト
ios/yakuqr-ios/yakuqr-iosTests/JAHISTests/JAHISValidatorTests.swift 変更: rawQRs型更新, バージョン名更新, 新テスト
ios/yakuqr-ios/yakuqr-iosTests/JAHISTests/ScannerViewModelTests.swift 変更: バージョン名更新
ios/yakuqr-ios/yakuqr-iosTests/JAHISTests/PrescriptionTextFormatterTests.swift 変更: 引数型更新, バージョン名更新
```

---

### Task 1: データモデル変更 + 全ファイルのコンパイル修正

Swift はビルド単位でコンパイルするため、`JAHISVersion` のケース名変更は全参照ファイルを一括修正する必要がある。

**Files:**
- Modify: `ios/yakuqr-ios/yakuqr-ios/JAHIS/JAHISModels.swift`
- Modify: `ios/yakuqr-ios/yakuqr-ios/JAHIS/JAHISParser.swift`
- Modify: `ios/yakuqr-ios/yakuqr-ios/JAHIS/JAHISValidator.swift`
- Modify: `ios/yakuqr-ios/yakuqr-ios/JAHIS/PrescriptionTextFormatter.swift`
- Modify: `ios/yakuqr-ios/yakuqr-ios/Features/Scanner/ScannerViewModel.swift`
- Modify: `ios/yakuqr-ios/yakuqr-iosTests/JAHISTests/JAHISParserTests.swift`
- Modify: `ios/yakuqr-ios/yakuqr-iosTests/JAHISTests/JAHISValidatorTests.swift`
- Modify: `ios/yakuqr-ios/yakuqr-iosTests/JAHISTests/ScannerViewModelTests.swift`
- Modify: `ios/yakuqr-ios/yakuqr-iosTests/JAHISTests/PrescriptionTextFormatterTests.swift`

- [ ] **Step 1: JAHISModels.swift を全面書き換え**

`ios/yakuqr-ios/yakuqr-ios/JAHIS/JAHISModels.swift` を以下に置き換える:

```swift
enum JAHISVersion: Int {
    case unknown = 0
    case v1_0 = 1
    case v1_1 = 2
    case v2_0 = 3
    case v2_1 = 4
    case v2_6 = 8

    var displayName: String {
        switch self {
        case .v1_0: return "Ver.1.0"
        case .v1_1: return "Ver.1.1"
        case .v2_0: return "Ver.2.0"
        case .v2_1: return "Ver.2.1"
        case .v2_6: return "Ver.2.6"
        case .unknown: return "Unknown"
        }
    }
}

struct RawQR {
    let text: String
    let errMsg: String
    var isSuccess: Bool { errMsg.isEmpty }
}

struct JAHISRecord {
    let type: String
    let fields: [String]
}

struct JAHISSplitInfo {
    let current: Int
    let total: Int
}

struct JAHISPrescription {
    let version: JAHISVersion
    let rawQRs: [RawQR]
    let records: [JAHISRecord]
    let recordMap: [String: [JAHISRecord]]
    let splitInfos: [JAHISSplitInfo]
}

enum ValidationLevel: Int, Comparable {
    case error = 0
    case warning = 1
    case info = 2

    static func < (lhs: ValidationLevel, rhs: ValidationLevel) -> Bool {
        lhs.rawValue < rhs.rawValue
    }

    var displayName: String {
        switch self {
        case .error: return "ERROR"
        case .warning: return "WARNING"
        case .info: return "INFO"
        }
    }
}

struct JAHISValidationResult {
    let level: ValidationLevel
    let field: String
    let message: String

    var description: String { "[\(level.displayName)] \(field): \(message)" }
}
```

- [ ] **Step 2: JAHISParser.swift のシグネチャ・バージョン参照を修正**

`parse()` の引数と `combineQRs` の先頭、`detectVersion` のバージョン参照を更新する。ファイル全体を以下に置き換える（911分割ロジックは Task 2 で追加するため、ここでは既存の連番分割のみ）:

```swift
import Foundation

struct JAHISParser {

    static func parse(_ rawQRs: [RawQR]) -> (JAHISPrescription, [String]) {
        var msgs: [String] = []
        let (combined, splitInfos, warns) = combineQRs(rawQRs)
        msgs.append(contentsOf: warns)

        let records = parseRecords(combined)
        var recordMap: [String: [JAHISRecord]] = [:]
        for r in records {
            recordMap[r.type, default: []].append(r)
        }

        let (version, info) = detectVersion(recordMap: recordMap, rawQRs: rawQRs)
        if let info { msgs.append(info) }

        let prescription = JAHISPrescription(
            version: version,
            rawQRs: rawQRs,
            records: records,
            recordMap: recordMap,
            splitInfos: splitInfos
        )
        return (prescription, msgs)
    }

    // MARK: - Private

    private struct QRPart {
        let content: String
        let seq: Int
    }

    private static func combineQRs(_ rawQRs: [RawQR]) -> (String, [JAHISSplitInfo], [String]) {
        var msgs: [String] = []
        var parts: [QRPart] = []
        var nonSplit: [String] = []

        let texts = rawQRs.filter { $0.isSuccess }.map { $0.text }

        for raw in texts {
            let lines = splitLines(raw)
            guard let firstLine = lines.first?.trimmingCharacters(in: .whitespaces) else { continue }

            if firstLine.hasPrefix("JAHISTC") {
                var seq = 1
                if let commaIdx = firstLine.lastIndex(of: ",") {
                    let seqStr = String(firstLine[firstLine.index(after: commaIdx)...])
                        .trimmingCharacters(in: .whitespaces)
                    if let n = Int(seqStr), n > 0 { seq = n }
                }
                parts.append(QRPart(content: lines.dropFirst().joined(separator: "\n"), seq: seq))
            } else {
                nonSplit.append(raw)
            }
        }

        if !parts.isEmpty {
            let maxSeq = parts.max(by: { $0.seq < $1.seq })!.seq
            var sorted = [String](repeating: "", count: maxSeq)
            for pt in parts where pt.seq >= 1 && pt.seq <= maxSeq {
                sorted[pt.seq - 1] = pt.content
            }
            for (i, s) in sorted.enumerated() where s.isEmpty {
                msgs.append("[WARNING] 分割QRの \(i + 1)/\(maxSeq) 枚目が見つかりません。取得済み分で処理を続行します")
            }
            let infos = parts.sorted { $0.seq < $1.seq }
                .map { JAHISSplitInfo(current: $0.seq, total: maxSeq) }
            var combined = sorted.joined(separator: "\n")
            if !nonSplit.isEmpty { combined += nonSplit.joined() }
            return (combined, infos, msgs)
        }

        return (nonSplit.joined(separator: "\n"), [], msgs)
    }

    private static func parseRecords(_ combined: String) -> [JAHISRecord] {
        splitLines(combined).compactMap { line -> JAHISRecord? in
            let trimmed = line.trimmingCharacters(in: .whitespaces)
            guard !trimmed.isEmpty else { return nil }
            let fields = trimmed.components(separatedBy: ",")
            return JAHISRecord(type: fields[0], fields: fields)
        }
    }

    private static func detectVersion(
        recordMap: [String: [JAHISRecord]],
        rawQRs: [RawQR]
    ) -> (JAHISVersion, String?) {
        for raw in rawQRs where raw.isSuccess {
            let lines = splitLines(raw.text)
            guard let first = lines.first?.trimmingCharacters(in: .whitespaces),
                  first.hasPrefix("JAHISTC") else { continue }
            let afterPrefix = String(first.dropFirst(7))
            let numPart = String(afterPrefix.split(separator: ",").first ?? "")
            let numStr = String(numPart.drop(while: { $0 == "0" }))
            if let n = Int(numStr) {
                switch n {
                case 1: return (.v1_0, nil)
                case 2: return (.v1_1, nil)
                case 3: return (.v2_0, nil)
                case 4: return (.v2_1, nil)
                case 8: return (.v2_6, nil)
                default: break
                }
            }
        }
        if let recs = recordMap["1"], let first = recs.first, first.fields.count >= 2 {
            switch first.fields[1] {
            case "1": return (.v1_0, nil)
            case "2": return (.v1_1, nil)
            case "3": return (.v2_0, nil)
            default: break
            }
        }
        return (.v2_1, "[INFO] バージョンを検出できなかったため、Ver.2.1（最新版）として処理します")
    }

    private static func splitLines(_ s: String) -> [String] {
        s.replacingOccurrences(of: "\r\n", with: "\n")
         .replacingOccurrences(of: "\r", with: "\n")
         .components(separatedBy: "\n")
    }
}
```

- [ ] **Step 3: JAHISValidator.swift のバージョン参照を修正**

`rulesFor` の分岐を新しいバージョン名に合わせ、`ver4Rules` → `newFormatRules`、`ver2ver3Rules` → `oldFormatRules` にリネームする。`validate()` は Task 3 で拡張するため今は変更しない。ファイル全体を以下に置き換える:

```swift
struct JAHISValidator {

    static func validate(_ prescription: JAHISPrescription) -> [JAHISValidationResult] {
        rulesFor(prescription.version).compactMap { rule -> JAHISValidationResult? in
            let (ok, msg) = rule.check(prescription)
            guard !ok else { return nil }
            return JAHISValidationResult(level: rule.level, field: rule.field, message: msg)
        }
    }

    // MARK: - Private

    private struct Rule {
        let field: String
        let level: ValidationLevel
        let check: (JAHISPrescription) -> (Bool, String)
    }

    private static func rulesFor(_ version: JAHISVersion) -> [Rule] {
        switch version {
        case .v2_1, .v2_6, .unknown:
            return newFormatRules()
        case .v1_0, .v1_1, .v2_0:
            return oldFormatRules(version)
        }
    }

    private static func newFormatRules() -> [Rule] {
        [
            Rule(field: "処方箋情報(レコード1)", level: .error) { p in
                guard p.recordMap["1"] != nil else {
                    return (false, "レコード種別1（処方箋情報）が存在しません")
                }
                return (true, "")
            },
            Rule(field: "患者氏名", level: .error) { p in
                guard let recs = p.recordMap["1"], let first = recs.first else { return (true, "") }
                guard first.fields.count >= 2, !first.fields[1].isEmpty else {
                    return (false, "患者氏名（レコード1 フィールド2）が空です")
                }
                return (true, "")
            },
            Rule(field: "患者生年月日", level: .error) { p in
                guard let recs = p.recordMap["1"], let first = recs.first else { return (true, "") }
                guard first.fields.count >= 4, !first.fields[3].isEmpty else {
                    return (false, "患者生年月日（レコード1 フィールド4）が空です")
                }
                guard isValidDate(first.fields[3]) else {
                    return (false, "患者生年月日のフォーマットが不正です（YYYYMMDD形式を期待）")
                }
                return (true, "")
            },
            Rule(field: "薬品情報(レコード201以上)", level: .warning) { p in
                let hasDrug = p.recordMap.keys.contains(where: { $0.count == 3 && $0.hasPrefix("2") })
                return hasDrug ? (true, "") : (false, "レコード種別201以上（薬品情報）が存在しません")
            },
        ]
    }

    private static func oldFormatRules(_ version: JAHISVersion) -> [Rule] {
        [
            Rule(field: "処方箋情報(レコード1)", level: .warning) { p in
                p.recordMap["1"] != nil ? (true, "") : (false, "レコード種別1（処方箋情報）が存在しません")
            },
            Rule(field: "患者情報(レコード2)", level: .error) { p in
                guard let recs = p.recordMap["2"], !recs.isEmpty else {
                    return (false, "レコード種別2（患者情報）が存在しません")
                }
                return (true, "")
            },
            Rule(field: "患者氏名", level: .error) { p in
                guard let recs = p.recordMap["2"], let first = recs.first else { return (true, "") }
                guard first.fields.count >= 2, !first.fields[1].isEmpty else {
                    return (false, "患者氏名（レコード2 フィールド2）が空です")
                }
                return (true, "")
            },
            Rule(field: "患者生年月日", level: .error) { p in
                guard let recs = p.recordMap["2"], let first = recs.first else { return (true, "") }
                guard first.fields.count >= 4, !first.fields[3].isEmpty else {
                    return (false, "患者生年月日（レコード2 フィールド4）が空です")
                }
                guard isValidDate(first.fields[3]) else {
                    return (false, "患者生年月日のフォーマットが不正です（YYYYMMDD形式を期待）")
                }
                return (true, "")
            },
            Rule(field: "薬品情報(レコード6)", level: .warning) { p in
                p.recordMap["6"] != nil ? (true, "") : (false, "レコード種別6（処方薬品情報）が存在しません")
            },
            Rule(field: "バージョン互換性", level: .info) { _ in
                (false, "Ver.2/Ver.3 形式を検出しました。一部のフィールドはVer.4と異なる場合があります")
            },
        ]
    }

    private static func isValidDate(_ s: String) -> Bool {
        s.count == 8 && s.allSatisfy(\.isNumber)
    }
}
```

- [ ] **Step 4: PrescriptionTextFormatter.swift の rawQRs 反復とバージョン判定を修正**

`rawQRs.enumerated()` のループで `raw` が `RawQR` 型になるため `.text` を参照する。`prescription.version == .v4` を新フォーマット判定に変更する。以下の 2 箇所を編集する:

```swift
// 変更前（rawQRs のループ）:
for (i, raw) in prescription.rawQRs.enumerated() {
    lines.append("[QR #\(i + 1)]")
    lines.append(raw)
    lines.append("")
}

// 変更後:
for (i, raw) in prescription.rawQRs.enumerated() {
    lines.append("[QR #\(i + 1)]")
    if raw.isSuccess {
        lines.append(raw.text)
    } else {
        lines.append("(読み取り失敗: \(raw.errMsg))")
    }
    lines.append("")
}
```

```swift
// 変更前（appendPatientInfo 内）:
if prescription.version == .v4 {

// 変更後:
if [JAHISVersion.v2_1, .v2_6, .unknown].contains(prescription.version) {
```

- [ ] **Step 5: ScannerViewModel.swift の型変更**

```swift
// 変更前:
@Published var scannedQRs: [String] = []

func addQR(_ value: String) {
    guard !seenQRs.contains(value) else { return }
    seenQRs.insert(value)
    scannedQRs.append(value)
}

// 変更後:
@Published var scannedQRs: [RawQR] = []

func addQR(_ value: String) {
    guard !seenQRs.contains(value) else { return }
    seenQRs.insert(value)
    scannedQRs.append(RawQR(text: value, errMsg: ""))
}
```

- [ ] **Step 6: JAHISParserTests.swift の全参照を更新**

ファイル全体を以下に置き換える:

```swift
import XCTest
@testable import yakuqr_ios

final class JAHISParserTests: XCTestCase {

    private func readTestdata(_ name: String) throws -> String {
        let thisFile = URL(fileURLWithPath: #file)
        let repoRoot = thisFile
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .deletingLastPathComponent()
        let path = repoRoot.appendingPathComponent("testdata/\(name)").path
        return try String(contentsOfFile: path, encoding: .utf8)
    }

    private func rawQR(_ name: String) throws -> RawQR {
        RawQR(text: try readTestdata(name), errMsg: "")
    }

    func testParse_singleQR_ver4_detectsVersion() throws {
        let (p, _) = JAHISParser.parse([try rawQR("ver4_single.txt")])
        XCTAssertEqual(p.version, .v2_1)
    }

    func testParse_singleQR_ver4_hasRecords() throws {
        let (p, _) = JAHISParser.parse([try rawQR("ver4_single.txt")])
        XCTAssertFalse(p.records.isEmpty)
        XCTAssertNotNil(p.recordMap["1"])
        XCTAssertNotNil(p.recordMap["201"])
    }

    func testParse_splitQR_combined() throws {
        let (p, _) = JAHISParser.parse([try rawQR("ver4_split_1.txt"), try rawQR("ver4_split_2.txt")])
        XCTAssertNotNil(p.recordMap["201"])
        XCTAssertEqual(p.splitInfos.count, 2)
    }

    func testParse_splitQR_missing_hasWarning() throws {
        let (_, msgs) = JAHISParser.parse([try rawQR("ver4_split_only2.txt")])
        XCTAssertTrue(msgs.contains(where: { $0.contains("分割") }))
    }

    func testParse_ver3_detectsVersion() throws {
        let (p, _) = JAHISParser.parse([try rawQR("ver3_single.txt")])
        XCTAssertEqual(p.version, .v2_0)
    }

    func testParse_ver2_detectsVersion() throws {
        let (p, _) = JAHISParser.parse([try rawQR("ver2_single.txt")])
        XCTAssertEqual(p.version, .v1_1)
    }

    func testParse_unknownVersion_fallsBackToVer2_1() {
        let raw = RawQR(text: "99,unknown\n2,テスト,テスト,19900101,1,,,", errMsg: "")
        let (p, msgs) = JAHISParser.parse([raw])
        XCTAssertEqual(p.version, .v2_1)
        XCTAssertTrue(msgs.contains(where: { $0.contains("Ver.2.1") }))
    }
}
```

- [ ] **Step 7: JAHISValidatorTests.swift の rawQRs 型と version 参照を更新**

ファイル全体を以下に置き換える:

```swift
import XCTest
@testable import yakuqr_ios

final class JAHISValidatorTests: XCTestCase {

    private func readTestdata(_ name: String) throws -> String {
        let thisFile = URL(fileURLWithPath: #file)
        let repoRoot = thisFile
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .deletingLastPathComponent()
        return try String(
            contentsOfFile: repoRoot.appendingPathComponent("testdata/\(name)").path,
            encoding: .utf8
        )
    }

    func testValidate_ver4_valid_hasNoErrors() throws {
        let raw = RawQR(text: try readTestdata("ver4_single.txt"), errMsg: "")
        let (p, _) = JAHISParser.parse([raw])
        let errors = JAHISValidator.validate(p).filter { $0.level == .error }
        XCTAssertTrue(errors.isEmpty, "有効なVer.4データにエラーがあってはならない: \(errors.map(\.description))")
    }

    func testValidate_ver4_missingRecord1_hasError() {
        let p = JAHISPrescription(
            version: .v2_1,
            rawQRs: [RawQR(text: "dummy", errMsg: "")],
            records: [],
            recordMap: [:],
            splitInfos: []
        )
        let results = JAHISValidator.validate(p)
        XCTAssertTrue(results.contains(where: { $0.level == .error && $0.field.contains("レコード1") }))
    }

    func testValidate_ver4_emptyPatientName_hasError() {
        let record1 = JAHISRecord(type: "1", fields: ["1", "", ""])
        let p = JAHISPrescription(
            version: .v2_1,
            rawQRs: [RawQR(text: "dummy", errMsg: "")],
            records: [record1],
            recordMap: ["1": [record1]],
            splitInfos: []
        )
        let results = JAHISValidator.validate(p)
        XCTAssertTrue(results.contains(where: { $0.level == .error && $0.field.contains("患者氏名") }))
    }

    func testValidate_ver4_invalidDateFormat_hasError() {
        let record1 = JAHISRecord(type: "1", fields: ["1", "山田太郎", "1", "INVALID"])
        let p = JAHISPrescription(
            version: .v2_1,
            rawQRs: [RawQR(text: "dummy", errMsg: "")],
            records: [record1],
            recordMap: ["1": [record1]],
            splitInfos: []
        )
        let results = JAHISValidator.validate(p)
        XCTAssertTrue(results.contains(where: { $0.level == .error && $0.field.contains("生年月日") }))
    }

    func testValidate_ver4_missingDrugRecord_hasWarning() {
        let record1 = JAHISRecord(type: "1", fields: ["1", "山田太郎", "1", "19700101"])
        let p = JAHISPrescription(
            version: .v2_1,
            rawQRs: [RawQR(text: "dummy", errMsg: "")],
            records: [record1],
            recordMap: ["1": [record1]],
            splitInfos: []
        )
        let results = JAHISValidator.validate(p)
        XCTAssertTrue(results.contains(where: { $0.level == .warning && $0.field.contains("薬品情報") }))
    }

    func testValidate_ver2_valid_hasNoErrors() throws {
        let raw = RawQR(text: try readTestdata("ver2_single.txt"), errMsg: "")
        let (p, _) = JAHISParser.parse([raw])
        let errors = JAHISValidator.validate(p).filter { $0.level == .error }
        XCTAssertTrue(errors.isEmpty, "有効なVer.2データにエラーがあってはならない: \(errors.map(\.description))")
    }
}
```

- [ ] **Step 8: ScannerViewModelTests.swift の version 参照を更新**

```swift
// 変更前:
XCTAssertEqual(vm.parseResult?.prescription.version, .v4)

// 変更後:
XCTAssertEqual(vm.parseResult?.prescription.version, .v2_1)
```

- [ ] **Step 9: PrescriptionTextFormatterTests.swift の型と version 参照を更新**

ファイル全体を以下に置き換える:

```swift
import XCTest
@testable import yakuqr_ios

final class PrescriptionTextFormatterTests: XCTestCase {

    private func readTestdata(_ name: String) throws -> String {
        let thisFile = URL(fileURLWithPath: #file)
        let repoRoot = thisFile
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .deletingLastPathComponent()
        return try String(
            contentsOfFile: repoRoot.appendingPathComponent("testdata/\(name)").path,
            encoding: .utf8
        )
    }

    func testFormat_ver4_containsRawQR() throws {
        let raw = RawQR(text: "JAHISTC04,1\n1,山田太郎,1,19700101,,,,,,\n201,1,アムロジピン錠,1,錠,4,xxx,1", errMsg: "")
        let (p, _) = JAHISParser.parse([raw])
        let text = PrescriptionTextFormatter.format(p, validations: JAHISValidator.validate(p))
        XCTAssertTrue(text.contains("RAW QR DATA"))
        XCTAssertTrue(text.contains("JAHISTC04"))
    }

    func testFormat_ver4_containsValidation() throws {
        let raw = RawQR(text: "JAHISTC04,1\n1,,,", errMsg: "")
        let (p, _) = JAHISParser.parse([raw])
        let text = PrescriptionTextFormatter.format(p, validations: JAHISValidator.validate(p))
        XCTAssertTrue(text.contains("VALIDATION"))
        XCTAssertTrue(text.contains("ERROR"))
    }

    func testFormat_ver4_noValidationErrors_containsOkMessage() {
        let record1 = JAHISRecord(type: "1", fields: ["1", "山田太郎", "1", "19700101"])
        let record201 = JAHISRecord(type: "201", fields: ["201", "1", "テスト薬", "1", "錠", "4", "xxx", "1"])
        let p = JAHISPrescription(
            version: .v2_1,
            rawQRs: [RawQR(text: "dummy", errMsg: "")],
            records: [record1, record201],
            recordMap: ["1": [record1], "201": [record201]],
            splitInfos: []
        )
        let text = PrescriptionTextFormatter.format(p, validations: [])
        XCTAssertTrue(text.contains("問題は検出されませんでした"))
    }

    func testFormat_ver2_containsPatientName() throws {
        let raw = RawQR(text: try readTestdata("ver2_single.txt"), errMsg: "")
        let (p, _) = JAHISParser.parse([raw])
        let text = PrescriptionTextFormatter.format(p, validations: [])
        XCTAssertTrue(text.contains("RAW QR DATA"))
        XCTAssertTrue(text.contains("Ver.1.1"))
        XCTAssertTrue(text.contains("山田次郎"))
    }
}
```

- [ ] **Step 10: ビルドしてコンパイルエラーがないことを確認**

```bash
xcodebuild build \
  -project ios/yakuqr-ios/yakuqr-ios.xcodeproj \
  -scheme yakuqr-ios \
  -destination 'platform=iOS Simulator,name=iPhone 17 Pro' \
  2>&1 | grep -E "error:|Build succeeded|Build FAILED"
```

期待: `Build succeeded`（`error:` が 0 件）

- [ ] **Step 11: テストを実行して全て通ることを確認**

```bash
xcodebuild test \
  -project ios/yakuqr-ios/yakuqr-ios.xcodeproj \
  -scheme yakuqr-ios \
  -destination 'platform=iOS Simulator,name=iPhone 17 Pro' \
  2>&1 | grep -E "Test Suite|passed|failed"
```

期待: 全テスト passed、failed 0

- [ ] **Step 12: コミット**

```bash
git add ios/yakuqr-ios/yakuqr-ios/JAHIS/JAHISModels.swift \
        ios/yakuqr-ios/yakuqr-ios/JAHIS/JAHISParser.swift \
        ios/yakuqr-ios/yakuqr-ios/JAHIS/JAHISValidator.swift \
        ios/yakuqr-ios/yakuqr-ios/JAHIS/PrescriptionTextFormatter.swift \
        ios/yakuqr-ios/yakuqr-ios/Features/Scanner/ScannerViewModel.swift \
        ios/yakuqr-ios/yakuqr-iosTests/JAHISTests/JAHISParserTests.swift \
        ios/yakuqr-ios/yakuqr-iosTests/JAHISTests/JAHISValidatorTests.swift \
        ios/yakuqr-ios/yakuqr-iosTests/JAHISTests/ScannerViewModelTests.swift \
        ios/yakuqr-ios/yakuqr-iosTests/JAHISTests/PrescriptionTextFormatterTests.swift
git commit -m "refactor(ios): RawQR導入・JAHISVersionリネーム・全ファイルのコンパイル修正"
```

---

### Task 2: パーサー — 911 累積型分割 + Ver.2.6 バージョン検出

**Files:**
- Modify: `ios/yakuqr-ios/yakuqr-iosTests/JAHISTests/JAHISParserTests.swift`
- Modify: `ios/yakuqr-ios/yakuqr-ios/JAHIS/JAHISParser.swift`

- [ ] **Step 1: 失敗するテストを追加する**

`JAHISParserTests.swift` のクラス末尾（最後の `}` の前）に追加:

```swift
func testParse_ver2_6_detectsVersion() throws {
    let raw = RawQR(text: try readTestdata("ver2_6_911split_1.txt"), errMsg: "")
    let (p, _) = JAHISParser.parse([raw])
    XCTAssertEqual(p.version, .v2_6)
}

func testParse_911split_3way_all_hasAllRps() throws {
    let raws = try [
        rawQR("ver2_6_911split3_1.txt"),
        rawQR("ver2_6_911split3_2.txt"),
        rawQR("ver2_6_911split3_3.txt"),
    ]
    let (p, _) = JAHISParser.parse(raws)
    XCTAssertEqual(p.version, .v2_6)
    XCTAssertEqual(p.recordMap["201"]?.count, 3, "Rp1〜Rp3の3件が必要")
    XCTAssertNil(p.recordMap["911"], "911行はRecordMapに残らない")
}

func testParse_911split_missingQR2_usesMaxCumulative() throws {
    // QR1+QR3のみ（QR2欠落）: 累積型のため QR3 がすべてのRpを含む
    let raws = try [rawQR("ver2_6_911split3_1.txt"), rawQR("ver2_6_911split3_3.txt")]
    let (p, _) = JAHISParser.parse(raws)
    XCTAssertEqual(p.recordMap["201"]?.count, 3, "累積型: QR2欠落でもRp1〜Rp3全て取得できる")
    XCTAssertNil(p.recordMap["911"])
}

func testParse_911split_qr1Only_has911InRecordMap() throws {
    // QR1のみ: 1パーツなので 911 パスに入らず、911行がRecordMapに残る
    let (p, _) = JAHISParser.parse([try rawQR("ver2_6_911split_1.txt")])
    XCTAssertNotNil(p.recordMap["911"], "QR1のみのとき911行がRecordMapに残る")
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
xcodebuild test \
  -project ios/yakuqr-ios/yakuqr-ios.xcodeproj \
  -scheme yakuqr-ios \
  -destination 'platform=iOS Simulator,name=iPhone 17 Pro' \
  -only-testing:yakuqr-iosTests/JAHISParserTests/testParse_ver2_6_detectsVersion \
  2>&1 | grep -E "passed|failed"
```

期待: `failed`（まだ v2_6 を検出できない）

- [ ] **Step 3: JAHISParser.swift に 911 分割ロジックを追加**

`JAHISParser.swift` の `QRPart` struct の後に `Part911` struct と 3 つのヘルパーを追加し、`combineQRs` に 911 分岐を挿入する。ファイル全体を以下に置き換える:

```swift
import Foundation

struct JAHISParser {

    static func parse(_ rawQRs: [RawQR]) -> (JAHISPrescription, [String]) {
        var msgs: [String] = []
        let (combined, splitInfos, warns) = combineQRs(rawQRs)
        msgs.append(contentsOf: warns)

        let records = parseRecords(combined)
        var recordMap: [String: [JAHISRecord]] = [:]
        for r in records {
            recordMap[r.type, default: []].append(r)
        }

        let (version, info) = detectVersion(recordMap: recordMap, rawQRs: rawQRs)
        if let info { msgs.append(info) }

        let prescription = JAHISPrescription(
            version: version,
            rawQRs: rawQRs,
            records: records,
            recordMap: recordMap,
            splitInfos: splitInfos
        )
        return (prescription, msgs)
    }

    // MARK: - Private

    private struct QRPart {
        let content: String
        let seq: Int
    }

    private struct Part911 {
        let content: String
        let dataID: String
        let total: Int
        let current: Int
    }

    private static func combineQRs(_ rawQRs: [RawQR]) -> (String, [JAHISSplitInfo], [String]) {
        var msgs: [String] = []
        var parts: [QRPart] = []
        var nonSplit: [String] = []

        let texts = rawQRs.filter { $0.isSuccess }.map { $0.text }

        for raw in texts {
            let lines = splitLines(raw)
            guard let firstLine = lines.first?.trimmingCharacters(in: .whitespaces) else { continue }

            if firstLine.hasPrefix("JAHISTC") {
                var seq = 1
                if let commaIdx = firstLine.lastIndex(of: ",") {
                    let seqStr = String(firstLine[firstLine.index(after: commaIdx)...])
                        .trimmingCharacters(in: .whitespaces)
                    if let n = Int(seqStr), n > 0 { seq = n }
                }
                parts.append(QRPart(content: lines.dropFirst().joined(separator: "\n"), seq: seq))
            } else {
                nonSplit.append(raw)
            }
        }

        if !parts.isEmpty {
            // 911累積型分割: 全パーツが同一seqかつ全パーツに911レコードがある場合
            if allSameSeq(parts), parts.count > 1, let sp = parse911Parts(parts) {
                let last = sp.max(by: { $0.current < $1.current })!
                var content = remove911Lines(last.content)
                if !nonSplit.isEmpty { content += nonSplit.joined() }
                let infos = sp.map { JAHISSplitInfo(current: $0.current, total: $0.total) }
                return (content, infos, msgs)
            }

            // JAHISTC連番分割
            let maxSeq = parts.max(by: { $0.seq < $1.seq })!.seq
            var sorted = [String](repeating: "", count: maxSeq)
            for pt in parts where pt.seq >= 1 && pt.seq <= maxSeq {
                sorted[pt.seq - 1] = pt.content
            }
            for (i, s) in sorted.enumerated() where s.isEmpty {
                msgs.append("[WARNING] 分割QRの \(i + 1)/\(maxSeq) 枚目が見つかりません。取得済み分で処理を続行します")
            }
            let infos = parts.sorted { $0.seq < $1.seq }
                .map { JAHISSplitInfo(current: $0.seq, total: maxSeq) }
            var combined = sorted.joined(separator: "\n")
            if !nonSplit.isEmpty { combined += nonSplit.joined() }
            return (combined, infos, msgs)
        }

        return (nonSplit.joined(separator: "\n"), [], msgs)
    }

    private static func allSameSeq(_ parts: [QRPart]) -> Bool {
        guard parts.count >= 2 else { return false }
        let first = parts[0].seq
        return parts.dropFirst().allSatisfy { $0.seq == first }
    }

    private static func parse911Parts(_ parts: [QRPart]) -> [Part911]? {
        var result: [Part911] = []
        for pt in parts {
            var found = false
            for line in splitLines(pt.content) {
                let trimmed = line.trimmingCharacters(in: .whitespaces)
                guard trimmed.hasPrefix("911,") else { continue }
                let fields = trimmed.components(separatedBy: ",")
                guard fields.count >= 4,
                      let total = Int(fields[2].trimmingCharacters(in: .whitespaces)),
                      let current = Int(fields[3].trimmingCharacters(in: .whitespaces)) else { continue }
                result.append(Part911(
                    content: pt.content,
                    dataID: fields[1].trimmingCharacters(in: .whitespaces),
                    total: total,
                    current: current
                ))
                found = true
                break
            }
            if !found { return nil }
        }
        return result.isEmpty ? nil : result
    }

    private static func remove911Lines(_ content: String) -> String {
        splitLines(content)
            .filter { !$0.trimmingCharacters(in: .whitespaces).hasPrefix("911,") }
            .joined(separator: "\n")
    }

    private static func parseRecords(_ combined: String) -> [JAHISRecord] {
        splitLines(combined).compactMap { line -> JAHISRecord? in
            let trimmed = line.trimmingCharacters(in: .whitespaces)
            guard !trimmed.isEmpty else { return nil }
            let fields = trimmed.components(separatedBy: ",")
            return JAHISRecord(type: fields[0], fields: fields)
        }
    }

    private static func detectVersion(
        recordMap: [String: [JAHISRecord]],
        rawQRs: [RawQR]
    ) -> (JAHISVersion, String?) {
        for raw in rawQRs where raw.isSuccess {
            let lines = splitLines(raw.text)
            guard let first = lines.first?.trimmingCharacters(in: .whitespaces),
                  first.hasPrefix("JAHISTC") else { continue }
            let afterPrefix = String(first.dropFirst(7))
            let numPart = String(afterPrefix.split(separator: ",").first ?? "")
            let numStr = String(numPart.drop(while: { $0 == "0" }))
            if let n = Int(numStr) {
                switch n {
                case 1: return (.v1_0, nil)
                case 2: return (.v1_1, nil)
                case 3: return (.v2_0, nil)
                case 4: return (.v2_1, nil)
                case 8: return (.v2_6, nil)
                default: break
                }
            }
        }
        if let recs = recordMap["1"], let first = recs.first, first.fields.count >= 2 {
            switch first.fields[1] {
            case "1": return (.v1_0, nil)
            case "2": return (.v1_1, nil)
            case "3": return (.v2_0, nil)
            default: break
            }
        }
        return (.v2_1, "[INFO] バージョンを検出できなかったため、Ver.2.1（最新版）として処理します")
    }

    private static func splitLines(_ s: String) -> [String] {
        s.replacingOccurrences(of: "\r\n", with: "\n")
         .replacingOccurrences(of: "\r", with: "\n")
         .components(separatedBy: "\n")
    }
}
```

- [ ] **Step 4: テストが通ることを確認**

```bash
xcodebuild test \
  -project ios/yakuqr-ios/yakuqr-ios.xcodeproj \
  -scheme yakuqr-ios \
  -destination 'platform=iOS Simulator,name=iPhone 17 Pro' \
  2>&1 | grep -E "Test Suite|passed|failed"
```

期待: 全テスト passed

- [ ] **Step 5: コミット**

```bash
git add ios/yakuqr-ios/yakuqr-ios/JAHIS/JAHISParser.swift \
        ios/yakuqr-ios/yakuqr-iosTests/JAHISTests/JAHISParserTests.swift
git commit -m "feat(ios): 911累積型分割パーサーとVer.2.6バージョン検出を追加"
```

---

### Task 3: バリデーター — 4 チェック追加

**Files:**
- Modify: `ios/yakuqr-ios/yakuqr-iosTests/JAHISTests/JAHISValidatorTests.swift`
- Modify: `ios/yakuqr-ios/yakuqr-ios/JAHIS/JAHISValidator.swift`

- [ ] **Step 1: 失敗するテストを追加する**

`JAHISValidatorTests.swift` のクラス末尾（最後の `}` の前）に追加:

```swift
func testValidate_911RecordPresent_returnsWarning() {
    let rec911 = JAHISRecord(type: "911", fields: ["911", "00000000000002", "3", "1"])
    let p = JAHISPrescription(
        version: .v2_6,
        rawQRs: [],
        records: [rec911],
        recordMap: ["911": [rec911]],
        splitInfos: []
    )
    let results = JAHISValidator.validate(p)
    XCTAssertTrue(
        results.contains { $0.level == .warning && $0.field == "分割制御レコード 911" },
        "911レコード残存はWARNING"
    )
}

func testValidate_orphanRecord_returnsWarning() {
    let bad = JAHISRecord(type: "abc", fields: ["abc", "garbage"])
    let p = JAHISPrescription(
        version: .v2_1,
        rawQRs: [],
        records: [bad],
        recordMap: [:],
        splitInfos: []
    )
    let results = JAHISValidator.validate(p)
    XCTAssertTrue(
        results.contains { $0.level == .warning && $0.field == "レコード種別" },
        "非数字レコード種別はWARNING"
    )
}

func testValidate_readFailure_returnsWarning() {
    let p = JAHISPrescription(
        version: .v2_1,
        rawQRs: [RawQR(text: "", errMsg: "decode failed")],
        records: [],
        recordMap: [:],
        splitInfos: []
    )
    let results = JAHISValidator.validate(p)
    XCTAssertTrue(
        results.contains { $0.level == .warning && $0.field == "QR #1" },
        "読み取り失敗QRはWARNING"
    )
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
xcodebuild test \
  -project ios/yakuqr-ios/yakuqr-ios.xcodeproj \
  -scheme yakuqr-ios \
  -destination 'platform=iOS Simulator,name=iPhone 17 Pro' \
  -only-testing:yakuqr-iosTests/JAHISValidatorTests/testValidate_911RecordPresent_returnsWarning \
  2>&1 | grep -E "passed|failed"
```

期待: `failed`（まだ checkSplit911Incomplete が存在しない）

- [ ] **Step 3: JAHISValidator.swift に 4 チェックを追加**

ファイル全体を以下に置き換える:

```swift
struct JAHISValidator {

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

    // MARK: - QR品質チェック

    private static func checkQRReadFailures(_ p: JAHISPrescription) -> [JAHISValidationResult] {
        p.rawQRs.enumerated().compactMap { i, raw in
            guard !raw.isSuccess else { return nil }
            return JAHISValidationResult(
                level: .warning,
                field: "QR #\(i + 1)",
                message: "読み取り失敗（\(raw.errMsg)）。このQRに含まれるデータが欠落しています"
            )
        }
    }

    private static func checkGarbledData(_ p: JAHISPrescription) -> [JAHISValidationResult] {
        var results: [JAHISValidationResult] = []
        var reported = Set<String>()
        for r in p.records {
            guard r.fields.contains(where: { $0.contains("\u{FFFD}") }) else { continue }
            let qrNum = findQRForRecord(r, rawQRs: p.rawQRs)
            let key = qrNum.map { "\(r.type):\($0)" } ?? r.type
            guard !reported.contains(key) else { continue }
            reported.insert(key)
            let field = qrNum.map { "QR #\($0), レコード種別 \(r.type)" } ?? "レコード種別 \(r.type)"
            results.append(JAHISValidationResult(
                level: .warning,
                field: field,
                message: "文字化けが検出されました（QRコード境界での Shift-JIS 文字の分断と思われます）"
            ))
        }
        return results
    }

    private static func findQRForRecord(_ r: JAHISRecord, rawQRs: [RawQR]) -> Int? {
        let line = r.fields.joined(separator: ",")
        for (i, raw) in rawQRs.enumerated() {
            guard raw.isSuccess else { continue }
            if raw.text.contains(line) { return i + 1 }
        }
        let typePrefix = r.type + ","
        for (i, raw) in rawQRs.enumerated() {
            guard raw.isSuccess else { continue }
            for rawLine in splitLines(raw.text) {
                guard rawLine.hasPrefix(typePrefix), rawLine.count > typePrefix.count else { continue }
                if line.hasPrefix(rawLine) { return i + 1 }
            }
        }
        return nil
    }

    private static func checkOrphanRecords(_ p: JAHISPrescription) -> [JAHISValidationResult] {
        p.records.compactMap { r in
            guard !isNumericType(r.type) else { return nil }
            return JAHISValidationResult(
                level: .warning,
                field: "レコード種別",
                message: "不正なレコード種別 \"\(r.type)\" が検出されました（QRコード欠落による断片データの可能性）"
            )
        }
    }

    private static func checkSplit911Incomplete(_ p: JAHISPrescription) -> [JAHISValidationResult] {
        guard p.recordMap["911"] != nil else { return [] }
        return [JAHISValidationResult(
            level: .warning,
            field: "分割制御レコード 911",
            message: "分割制御レコード（911）が検出されました。分割QRの一部が未取得の可能性があります"
        )]
    }

    // MARK: - バージョン固有ルール

    private struct Rule {
        let field: String
        let level: ValidationLevel
        let check: (JAHISPrescription) -> (Bool, String)
    }

    private static func rulesFor(_ version: JAHISVersion) -> [Rule] {
        switch version {
        case .v2_1, .v2_6, .unknown:
            return newFormatRules()
        case .v1_0, .v1_1, .v2_0:
            return oldFormatRules(version)
        }
    }

    private static func newFormatRules() -> [Rule] {
        [
            Rule(field: "処方箋情報(レコード1)", level: .error) { p in
                guard p.recordMap["1"] != nil else {
                    return (false, "レコード種別1（処方箋情報）が存在しません")
                }
                return (true, "")
            },
            Rule(field: "患者氏名", level: .error) { p in
                guard let recs = p.recordMap["1"], let first = recs.first else { return (true, "") }
                guard first.fields.count >= 2, !first.fields[1].isEmpty else {
                    return (false, "患者氏名（レコード1 フィールド2）が空です")
                }
                return (true, "")
            },
            Rule(field: "患者生年月日", level: .error) { p in
                guard let recs = p.recordMap["1"], let first = recs.first else { return (true, "") }
                guard first.fields.count >= 4, !first.fields[3].isEmpty else {
                    return (false, "患者生年月日（レコード1 フィールド4）が空です")
                }
                guard isValidDate(first.fields[3]) else {
                    return (false, "患者生年月日のフォーマットが不正です（YYYYMMDD形式を期待）")
                }
                return (true, "")
            },
            Rule(field: "薬品情報(レコード201以上)", level: .warning) { p in
                let hasDrug = p.recordMap.keys.contains(where: { $0.count == 3 && $0.hasPrefix("2") })
                return hasDrug ? (true, "") : (false, "レコード種別201以上（薬品情報）が存在しません")
            },
        ]
    }

    private static func oldFormatRules(_ version: JAHISVersion) -> [Rule] {
        [
            Rule(field: "処方箋情報(レコード1)", level: .warning) { p in
                p.recordMap["1"] != nil ? (true, "") : (false, "レコード種別1（処方箋情報）が存在しません")
            },
            Rule(field: "患者情報(レコード2)", level: .error) { p in
                guard let recs = p.recordMap["2"], !recs.isEmpty else {
                    return (false, "レコード種別2（患者情報）が存在しません")
                }
                return (true, "")
            },
            Rule(field: "患者氏名", level: .error) { p in
                guard let recs = p.recordMap["2"], let first = recs.first else { return (true, "") }
                guard first.fields.count >= 2, !first.fields[1].isEmpty else {
                    return (false, "患者氏名（レコード2 フィールド2）が空です")
                }
                return (true, "")
            },
            Rule(field: "患者生年月日", level: .error) { p in
                guard let recs = p.recordMap["2"], let first = recs.first else { return (true, "") }
                guard first.fields.count >= 4, !first.fields[3].isEmpty else {
                    return (false, "患者生年月日（レコード2 フィールド4）が空です")
                }
                guard isValidDate(first.fields[3]) else {
                    return (false, "患者生年月日のフォーマットが不正です（YYYYMMDD形式を期待）")
                }
                return (true, "")
            },
            Rule(field: "薬品情報(レコード6)", level: .warning) { p in
                p.recordMap["6"] != nil ? (true, "") : (false, "レコード種別6（処方薬品情報）が存在しません")
            },
            Rule(field: "バージョン互換性", level: .info) { _ in
                (false, "Ver.2/Ver.3 形式を検出しました。一部のフィールドはVer.4と異なる場合があります")
            },
        ]
    }

    private static func isNumericType(_ s: String) -> Bool {
        !s.isEmpty && s.allSatisfy { $0 >= "0" && $0 <= "9" }
    }

    private static func isValidDate(_ s: String) -> Bool {
        s.count == 8 && s.allSatisfy(\.isNumber)
    }

    private static func splitLines(_ s: String) -> [String] {
        s.replacingOccurrences(of: "\r\n", with: "\n")
         .replacingOccurrences(of: "\r", with: "\n")
         .components(separatedBy: "\n")
    }
}
```

- [ ] **Step 4: 全テストが通ることを確認**

```bash
xcodebuild test \
  -project ios/yakuqr-ios/yakuqr-ios.xcodeproj \
  -scheme yakuqr-ios \
  -destination 'platform=iOS Simulator,name=iPhone 17 Pro' \
  2>&1 | grep -E "Test Suite|passed|failed"
```

期待: 全テスト passed、failed 0

- [ ] **Step 5: コミット**

```bash
git add ios/yakuqr-ios/yakuqr-ios/JAHIS/JAHISValidator.swift \
        ios/yakuqr-ios/yakuqr-iosTests/JAHISTests/JAHISValidatorTests.swift
git commit -m "feat(ios): バリデーター4チェック追加（QR品質・文字化け・不正レコード・911残存）"
```

- [ ] **Step 6: ブランチをプッシュする**

```bash
git push origin feat/swift-go-sync
```
