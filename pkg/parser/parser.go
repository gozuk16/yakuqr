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

	version, info := detectVersion(p.RecordMap)
	p.Version = version
	if info != "" {
		msgs = append(msgs, info)
	}

	return p, msgs
}

// combineQRs は分割QRを結合して単一のレコード文字列を返す。
func combineQRs(rawQRs []string) (string, []SplitInfo, []string) {
	var msgs []string

	type qrPart struct {
		content string
		info    SplitInfo
	}

	var parts []qrPart
	var nonSplit []string

	for _, raw := range rawQRs {
		lines := splitLines(raw)
		if len(lines) == 0 {
			continue
		}
		// 先頭レコードが種別9なら分割QR
		// フォーマット: "9,<ver>,<N>,<M>"
		// NOTE: JAHIS規約原文でフィールド位置を確認すること
		fields := strings.Split(lines[0], ",")
		if fields[0] == "9" && len(fields) >= 4 {
			n, _ := strconv.Atoi(fields[2])
			m, _ := strconv.Atoi(fields[3])
			parts = append(parts, qrPart{
				content: strings.Join(lines[1:], "\n"),
				info:    SplitInfo{Current: n, Total: m},
			})
		} else {
			nonSplit = append(nonSplit, raw)
		}
	}

	if len(parts) > 0 {
		total := parts[0].info.Total
		sorted := make([]string, total)

		// 番号順にソート
		sort.Slice(parts, func(i, j int) bool {
			return parts[i].info.Current < parts[j].info.Current
		})

		for _, pt := range parts {
			if pt.info.Current >= 1 && pt.info.Current <= total {
				sorted[pt.info.Current-1] = pt.content
			}
		}

		for i, s := range sorted {
			if s == "" {
				msgs = append(msgs, fmt.Sprintf("[WARNING] 分割QRの %d/%d 枚目が見つかりません。取得済み分で処理を続行します", i+1, total))
			}
		}

		var infos []SplitInfo
		for _, pt := range parts {
			infos = append(infos, pt.info)
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

// detectVersion はレコードマップからJAHISバージョンを検出する。
// レコード種別1の2番目フィールド（バージョン番号）を参照する。
// NOTE: バージョンフィールドの位置はJAHIS規約原文で確認すること。
func detectVersion(rm map[string][]Record) (Version, string) {
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
