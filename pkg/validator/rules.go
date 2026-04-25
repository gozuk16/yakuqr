package validator

import "github.com/gozuk16/yakuqr/pkg/parser"

// rule はバリデーションルール1件。
type rule struct {
	field string
	level Level
	check func(p parser.Prescription) (bool, string) // true=OK
}

// rulesFor はバージョンに応じたルールセットを返す。
// NOTE: 以下のルールはJAHIS規約の主要チェック項目のサンプル。
// 実装時に規約原文のVer.2〜Ver.4の必須/推奨フィールド定義を参照して拡充すること。
func rulesFor(v parser.Version) []rule {
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
					return true, "" // レコード2未存在はレコード2ルールで検出済み
				}
				fields := recs[0].Fields
				// 患者氏名はレコード2のフィールド2（index 1）
				// NOTE: 正確なフィールド番号はJAHIS規約原文で確認すること
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
				// 生年月日はレコード2のフィールド4（index 3）
				// NOTE: 正確なフィールド番号はJAHIS規約原文で確認すること
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

	// Ver.2/Ver.3 固有の注意事項
	if v == parser.Version2 || v == parser.Version3 {
		base = append(base, rule{
			field: "バージョン互換性",
			level: LevelInfo,
			check: func(p parser.Prescription) (bool, string) {
				return false, "Ver.2/Ver.3 形式を検出しました。一部のフィールドはVer.4と異なる場合があります"
			},
		})
	}

	return base
}

// isValidDate は YYYYMMDD 形式の8桁数字を検証する。
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
