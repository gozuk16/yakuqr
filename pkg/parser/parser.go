package parser

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Parse はQRコードのUTF-8文字列リストを受け取り、Prescriptionを返す。
// 第2戻り値はWARNING/INFOメッセージのリスト。
func Parse(rawQRs []string) (Prescription, []string) {
	var msgs []string
	p := Prescription{
		RawQRs:    rawQRs,
		RecordMap: make(map[string][]Record),
	}

	combined, splitInfos, warns := combineQRs(rawQRs)
	msgs = append(msgs, warns...)
	p.SplitInfos = splitInfos

	p.Records = parseRecords(combined)
	for _, r := range p.Records {
		p.RecordMap[r.Type] = append(p.RecordMap[r.Type], r)
	}

	version, info := detectVersion(p.RecordMap, rawQRs)
	p.Version = version
	if info != "" {
		msgs = append(msgs, info)
	}

	return p, msgs
}

// combineQRs は分割QRを結合して単一のレコード文字列を返す。
// JAHIS形式: 各QRの先頭行が "JAHISTC{ver},{seq}" (例: "JAHISTC04,1")
func combineQRs(rawQRs []string) (string, []SplitInfo, []string) {
	var msgs []string

	type qrPart struct {
		content string
		seq     int
	}

	var parts []qrPart
	var nonSplit []string

	for _, raw := range rawQRs {
		lines := splitLines(raw)
		if len(lines) == 0 {
			continue
		}
		firstLine := strings.TrimSpace(lines[0])
		if strings.HasPrefix(firstLine, "JAHISTC") {
			// "JAHISTC{ver},{seq}" から連番を取り出す
			seq := 1
			if idx := strings.LastIndex(firstLine, ","); idx >= 0 {
				if n, err := strconv.Atoi(strings.TrimSpace(firstLine[idx+1:])); err == nil && n > 0 {
					seq = n
				}
			}
			parts = append(parts, qrPart{
				content: strings.Join(lines[1:], "\n"),
				seq:     seq,
			})
		} else {
			nonSplit = append(nonSplit, raw)
		}
	}

	if len(parts) > 0 {
		// 連番の最大値を総数とする
		maxSeq := 0
		for _, pt := range parts {
			if pt.seq > maxSeq {
				maxSeq = pt.seq
			}
		}

		sort.Slice(parts, func(i, j int) bool {
			return parts[i].seq < parts[j].seq
		})

		sorted := make([]string, maxSeq)
		for _, pt := range parts {
			if pt.seq >= 1 && pt.seq <= maxSeq {
				sorted[pt.seq-1] = pt.content
			}
		}

		for i, s := range sorted {
			if s == "" {
				msgs = append(msgs, fmt.Sprintf("[WARNING] 分割QRの %d/%d 枚目が見つかりません。取得済み分で処理を続行します", i+1, maxSeq))
			}
		}

		var infos []SplitInfo
		for _, pt := range parts {
			infos = append(infos, SplitInfo{Current: pt.seq, Total: maxSeq})
		}
		return strings.Join(sorted, "\n"), infos, msgs
	}

	return strings.Join(nonSplit, "\n"), nil, msgs
}

// parseRecords は結合済みレコード文字列を[]Recordに変換する。
func parseRecords(combined string) []Record {
	var records []Record
	for _, line := range splitLines(combined) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Split(line, ",")
		records = append(records, Record{
			Type:   fields[0],
			Fields: fields,
		})
	}
	return records
}

// detectVersion はJAHISバージョンを検出する。
// Ver.4: JAHISTC ヘッダの2桁数字 ("04") を優先する。
// Ver.2/Ver.3: ヘッダなしの場合、レコード種別1のフィールド2 ("2"/"3") を参照する。
func detectVersion(rm map[string][]Record, rawQRs []string) (Version, string) {
	for _, raw := range rawQRs {
		lines := splitLines(raw)
		if len(lines) == 0 {
			continue
		}
		first := strings.TrimSpace(lines[0])
		if strings.HasPrefix(first, "JAHISTC") && len(first) >= 9 {
			switch first[7:9] {
			case "02":
				return Version2, ""
			case "03":
				return Version3, ""
			case "04":
				return Version4, ""
			}
		}
	}
	if recs, ok := rm["1"]; ok && len(recs) > 0 {
		fields := recs[0].Fields
		if len(fields) >= 2 {
			switch fields[1] {
			case "2":
				return Version2, ""
			case "3":
				return Version3, ""
			case "4":
				return Version4, ""
			}
		}
	}
	return Version4, "[INFO] バージョンを検出できなかったため、Ver.4（最新版）として処理します"
}

// splitLines は改行コード（CR+LF / LF / CR）でテキストを分割する。
func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Split(s, "\n")
}
