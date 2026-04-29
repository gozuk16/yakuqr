import Foundation

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
                (false, "Ver.1.x/Ver.2.0 形式を検出しました。一部のフィールドはVer.2.1と異なる場合があります")
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
