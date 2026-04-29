struct PrescriptionTextFormatter {

    static func format(
        _ prescription: JAHISPrescription,
        validations: [JAHISValidationResult]
    ) -> String {
        var lines: [String] = []

        lines.append("=== RAW QR DATA ===")
        for (i, raw) in prescription.rawQRs.enumerated() {
            lines.append("[QR #\(i + 1)]")
            if raw.isSuccess {
                lines.append(raw.text)
            } else {
                lines.append("(読み取り失敗: \(raw.errMsg))")
            }
            lines.append("")
        }

        lines.append("=== PARSED DATA ===")
        lines.append("バージョン: \(prescription.version.displayName)")

        appendPatientInfo(prescription, to: &lines)
        appendPrescriptionInfo(prescription, to: &lines)
        appendDrugInfo(prescription, to: &lines)

        lines.append("")
        lines.append("=== VALIDATION RESULTS ===")
        if validations.isEmpty {
            lines.append("問題は検出されませんでした")
        } else {
            for r in validations {
                lines.append(r.description)
            }
        }

        return lines.joined(separator: "\n")
    }

    // MARK: - Private

    private static func appendPatientInfo(_ prescription: JAHISPrescription, to lines: inout [String]) {
        // Ver.4: レコード1に患者情報
        if let recs = prescription.recordMap["1"], let first = recs.first {
            let f = first.fields
            // Ver.4 の場合、fields[1] は氏名（Ver.2/3 では fields[1] はバージョン番号）
            if [JAHISVersion.v2_1, .v2_6, .unknown].contains(prescription.version) {
                lines.append("--- 患者情報 ---")
                if f.count > 1 { lines.append("氏名: \(f[1])") }
                if f.count > 3 { lines.append("生年月日: \(formatDate(f[3]))") }
                return
            }
        }
        // Ver.2/3: レコード2に患者情報
        if let recs = prescription.recordMap["2"], let first = recs.first {
            lines.append("--- 患者情報 ---")
            let f = first.fields
            if f.count > 1 { lines.append("氏名: \(f[1])") }
            if f.count > 2 { lines.append("カナ名: \(f[2])") }
            if f.count > 3 { lines.append("生年月日: \(formatDate(f[3]))") }
            if f.count > 4 { lines.append("性別: \(formatSex(f[4]))") }
        }
    }

    private static func appendPrescriptionInfo(_ prescription: JAHISPrescription, to lines: inout [String]) {
        guard let recs = prescription.recordMap["1"], let first = recs.first else { return }
        let f = first.fields
        // Ver.4: fields[2] は医療機関コード / Ver.2,3: fields[2] は医療機関コード
        lines.append("--- 処方箋情報 ---")
        if f.count > 2 { lines.append("医療機関コード: \(f[2])") }
    }

    private static func appendDrugInfo(_ prescription: JAHISPrescription, to lines: inout [String]) {
        // Ver.4: レコード201以上（薬品情報）
        let ver4DrugRecords = prescription.records.filter {
            $0.type.count == 3 && $0.type.hasPrefix("2")
        }
        if !ver4DrugRecords.isEmpty {
            lines.append("--- 薬品情報 ---")
            for rec in ver4DrugRecords {
                let f = rec.fields
                let name = f.count > 2 ? f[2] : ""
                let dose = f.count > 3 ? f[3] : ""
                let unit = f.count > 4 ? f[4] : ""
                lines.append("薬品: \(name) \(dose)\(unit)")
            }
            return
        }
        // Ver.2/3: レコード6（薬品情報）
        if let recs = prescription.recordMap["6"] {
            lines.append("--- Rp情報 ---")
            for (i, rec) in recs.enumerated() {
                let f = rec.fields
                let name = f.count > 2 ? f[2] : ""
                let dose = f.count > 3 ? f[3] : ""
                let unit = f.count > 4 ? f[4] : ""
                let days = f.count > 6 ? f[6] : ""
                lines.append("Rp\(i + 1): \(name) \(dose)\(unit) \(days)日分")
            }
        }
    }

    private static func formatDate(_ s: String) -> String {
        guard s.count == 8 else { return s }
        let y = s.prefix(4)
        let m = s.dropFirst(4).prefix(2)
        let d = s.dropFirst(6)
        return "\(y)-\(m)-\(d)"
    }

    private static func formatSex(_ s: String) -> String {
        switch s {
        case "1": return "男"
        case "2": return "女"
        default: return s
        }
    }
}
