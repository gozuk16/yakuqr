package validator

import "github.com/gozuk16/yakuqr/pkg/parser"

type rule struct {
	field string
	level Level
	check func(p parser.Prescription) (bool, string)
}

func rulesFor(v parser.Version) []rule {
	if v == parser.VersionUnknown {
		return unknownVersionRules()
	}
	if v >= parser.Version2_1 {
		return ver4Rules()
	}
	return ver2ver3Rules(v)
}

func unknownVersionRules() []rule {
	return []rule{{
		field: "バージョン",
		level: LevelError,
		check: func(p parser.Prescription) (bool, string) {
			return false, "バージョンを検出できませんでした。QRコードの形式を確認してください"
		},
	}}
}

// ver4Rules は JAHIS Ver.4 (JAHISTC04 形式) のルールセット。
// Ver.4 では患者情報がレコード1に、薬品情報がレコード201以上に格納される。
func ver4Rules() []rule {
	return []rule{
		{
			field: "処方箋情報(レコード1)",
			level: LevelError,
			check: func(p parser.Prescription) (bool, string) {
				if _, ok := p.RecordMap["1"]; !ok {
					return false, "レコード種別1（処方箋情報）が存在しません"
				}
				return true, ""
			},
		},
		{
			field: "患者氏名",
			level: LevelError,
			check: func(p parser.Prescription) (bool, string) {
				recs, ok := p.RecordMap["1"]
				if !ok || len(recs) == 0 {
					return true, ""
				}
				fields := recs[0].Fields
				if len(fields) < 2 || fields[1] == "" {
					return false, "患者氏名（レコード1 フィールド2）が空です"
				}
				return true, ""
			},
		},
		{
			field: "患者生年月日",
			level: LevelError,
			check: func(p parser.Prescription) (bool, string) {
				recs, ok := p.RecordMap["1"]
				if !ok || len(recs) == 0 {
					return true, ""
				}
				fields := recs[0].Fields
				if len(fields) < 4 || fields[3] == "" {
					return false, "患者生年月日（レコード1 フィールド4）が空です"
				}
				if !isValidDate(fields[3]) {
					return false, "患者生年月日のフォーマットが不正です（YYYYMMDD形式を期待）"
				}
				return true, ""
			},
		},
		{
			field: "薬品情報(レコード201以上)",
			level: LevelWarning,
			check: func(p parser.Prescription) (bool, string) {
				for k := range p.RecordMap {
					if len(k) == 3 && k[0] == '2' {
						return true, ""
					}
				}
				return false, "レコード種別201以上（薬品情報）が存在しません"
			},
		},
	}
}

// ver2ver3Rules は JAHIS Ver.2/Ver.3 のルールセット。
// Ver.2/Ver.3 では患者情報がレコード2に、薬品情報がレコード6に格納される。
func ver2ver3Rules(v parser.Version) []rule {
	base := []rule{
		{
			field: "処方箋情報(レコード1)",
			level: LevelError,
			check: func(p parser.Prescription) (bool, string) {
				if _, ok := p.RecordMap["1"]; !ok {
					return false, "レコード種別1（処方箋情報）が存在しません"
				}
				return true, ""
			},
		},
		{
			field: "患者情報(レコード2)",
			level: LevelError,
			check: func(p parser.Prescription) (bool, string) {
				recs, ok := p.RecordMap["2"]
				if !ok || len(recs) == 0 {
					return false, "レコード種別2（患者情報）が存在しません"
				}
				return true, ""
			},
		},
		{
			field: "患者氏名",
			level: LevelError,
			check: func(p parser.Prescription) (bool, string) {
				recs, ok := p.RecordMap["2"]
				if !ok || len(recs) == 0 {
					return true, ""
				}
				fields := recs[0].Fields
				if len(fields) < 2 || fields[1] == "" {
					return false, "患者氏名（レコード2 フィールド2）が空です"
				}
				return true, ""
			},
		},
		{
			field: "患者生年月日",
			level: LevelError,
			check: func(p parser.Prescription) (bool, string) {
				recs, ok := p.RecordMap["2"]
				if !ok || len(recs) == 0 {
					return true, ""
				}
				fields := recs[0].Fields
				if len(fields) < 4 || fields[3] == "" {
					return false, "患者生年月日（レコード2 フィールド4）が空です"
				}
				if !isValidDate(fields[3]) {
					return false, "患者生年月日のフォーマットが不正です（YYYYMMDD形式を期待）"
				}
				return true, ""
			},
		},
		{
			field: "薬品情報(レコード6)",
			level: LevelWarning,
			check: func(p parser.Prescription) (bool, string) {
				if _, ok := p.RecordMap["6"]; !ok {
					return false, "レコード種別6（処方薬品情報）が存在しません"
				}
				return true, ""
			},
		},
	}

	if v <= parser.Version2_0 {
		base = append(base, rule{
			field: "バージョン互換性",
			level: LevelInfo,
			check: func(p parser.Prescription) (bool, string) {
				return false, "JAHISTC01〜03形式（Ver.1.0〜2.0）を検出しました。一部のフィールドはVer.2.1以降と異なる場合があります"
			},
		})
	}

	return base
}

func isValidDate(s string) bool {
	if len(s) != 8 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
