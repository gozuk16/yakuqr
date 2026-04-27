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
        version == .v4 ? ver4Rules() : ver2ver3Rules(version)
    }

    private static func ver4Rules() -> [Rule] {
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

    private static func ver2ver3Rules(_ version: JAHISVersion) -> [Rule] {
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
