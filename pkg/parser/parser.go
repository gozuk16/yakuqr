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

type qrPart struct {
	content string
	seq     int
}

// combineQRs は分割QRを結合して単一のレコード文字列を返す。
// JAHIS形式: 各QRの先頭行が "JAHISTC{ver},{seq}" (例: "JAHISTC04,1")
// 911分割: 全QRが同一seqかつ末尾に 911 レコードを持つ累積分割形式。
func combineQRs(rawQRs []string) (string, []SplitInfo, []string) {
	var msgs []string
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
		// 911分割: 全QRが同一 JAHISTC seq（通常すべて1）で 911レコードを持つ場合。
		// 累積型のため最も連番が大きいQRの本文を採用し、911行を除去する。
		if allSameSeq(parts) && len(parts) > 1 {
			if sp, ok := parse911Parts(parts); ok {
				sort.Slice(sp, func(i, j int) bool { return sp[i].current < sp[j].current })
				last := sp[len(sp)-1]
				content := remove911Lines(last.content)
				if len(nonSplit) > 0 {
					content += strings.Join(nonSplit, "")
				}
				var infos []SplitInfo
				for _, s := range sp {
					infos = append(infos, SplitInfo{Current: s.current, Total: s.total})
				}
				return content, infos, msgs
			}
		}

		// JAHISTC連番分割: 各QRが異なる seq を持つ通常の分割形式。
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

// allSameSeq は全パーツが同じ seq 値を持つかどうかを返す。
func allSameSeq(parts []qrPart) bool {
	if len(parts) < 2 {
		return false
	}
	first := parts[0].seq
	for _, p := range parts[1:] {
		if p.seq != first {
			return false
		}
	}
	return true
}

type part911 struct {
	content string
	dataID  string
	total   int
	current int
}

// parse911Parts は各パーツから 911 レコードを探す。
// 全パーツに 911 レコードがあれば part911 スライスを返す。
// 1つでも 911 がないパーツがあれば false を返す。
func parse911Parts(parts []qrPart) ([]part911, bool) {
	var result []part911
	for _, pt := range parts {
		found := false
		for _, line := range splitLines(pt.content) {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "911,") {
				continue
			}
			fields := strings.Split(line, ",")
			if len(fields) < 4 {
				continue
			}
			total, err1 := strconv.Atoi(strings.TrimSpace(fields[2]))
			current, err2 := strconv.Atoi(strings.TrimSpace(fields[3]))
			if err1 != nil || err2 != nil {
				continue
			}
			result = append(result, part911{
				content: pt.content,
				dataID:  strings.TrimSpace(fields[1]),
				total:   total,
				current: current,
			})
			found = true
			break
		}
		if !found {
			return nil, false
		}
	}
	return result, len(result) > 0
}

// remove911Lines は文字列から 911 レコード行を除去して返す。
func remove911Lines(content string) string {
	var kept []string
	for _, line := range splitLines(content) {
		if !strings.HasPrefix(strings.TrimSpace(line), "911,") {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
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
