package parser

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Parse はQRコードの結果リストを受け取り、Prescriptionを返す。
// 読み取り失敗エントリ（ErrMsg 非空）はパース対象から除外され、表示用に保持される。
// 第2戻り値はWARNING/INFOメッセージのリスト。
func Parse(rawQRs []RawQR) (Prescription, []string) {
	var msgs []string
	p := Prescription{
		RawQRs:    rawQRs,
		RecordMap: make(map[string][]Record),
	}

	// 成功したQRのテキストのみパース対象とする
	texts := make([]string, 0, len(rawQRs))
	for _, r := range rawQRs {
		if r.ErrMsg == "" {
			texts = append(texts, r.Text)
		}
	}

	combined, splitInfos, warns := combineQRs(texts)
	msgs = append(msgs, warns...)
	p.SplitInfos = splitInfos

	p.Records = parseRecords(combined)
	for _, r := range p.Records {
		p.RecordMap[r.Type] = append(p.RecordMap[r.Type], r)
	}

	version, info := detectVersion(p.RecordMap, texts)
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

		combined := strings.Join(sorted, "\n")
		// JAHISTC ヘッダーを持たない SA 継続QRはバイト境界で分割されているため
		// セパレータなしで直結する（\n を挿入すると CSV レコードが壊れる）。
		if len(nonSplit) > 0 {
			combined += strings.Join(nonSplit, "")
		}
		return combined, infos, msgs
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
// JAHISTC ヘッダーがあれば番号から直接 Version(n) を返す（1桁・2桁どちらも対応）。
// ヘッダーなしの旧フォーマットはレコード種別1のフィールド2（バージョン番号文字）を参照する。
// どちらでも検出できない場合は VersionUnknown と ERROR メッセージを返す。
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
	// ヘッダーなし旧フォーマット: レコード種別1のフィールド[1]がバージョン番号文字列
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

// splitLines は改行コード（CR+LF / LF / CR）でテキストを分割する。
func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Split(s, "\n")
}
