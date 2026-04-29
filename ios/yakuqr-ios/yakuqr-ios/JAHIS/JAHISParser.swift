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
            if !nonSplit.isEmpty { combined += "\n" + nonSplit.joined(separator: "\n") }
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
