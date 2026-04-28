package validator

import (
	"fmt"
	"strings"

	"github.com/gozuk16/yakuqr/pkg/parser"
)

// checkQRReadFailures は RawQRs に読み取り失敗エントリがあれば WARNING を返す。
func checkQRReadFailures(p parser.Prescription) []ValidationResult {
	var results []ValidationResult
	for i, r := range p.RawQRs {
		if r.ErrMsg != "" {
			results = append(results, ValidationResult{
				Level:   LevelWarning,
				Field:   fmt.Sprintf("QR #%d", i+1),
				Message: fmt.Sprintf("読み取り失敗（%s）。このQRに含まれるデータが欠落しています", r.ErrMsg),
			})
		}
	}
	return results
}

// checkGarbledData はレコードフィールドに Unicode 置換文字（U+FFFD）が含まれる場合
// WARNING を返す。Shift-JIS 文字が QR 分割境界で切断されると発生する。
// 同一レコード種別かつ同一QR番号の組み合わせは1件のみ報告する。
func checkGarbledData(p parser.Prescription) []ValidationResult {
	var results []ValidationResult
	reported := make(map[string]bool)
	for _, r := range p.Records {
		for _, f := range r.Fields {
			if strings.ContainsRune(f, '�') {
				qrNum, found := findQRForRecord(r, p.RawQRs)
				key := r.Type
				if found {
					key = fmt.Sprintf("%s:%d", r.Type, qrNum)
				}
				if reported[key] {
					break
				}
				reported[key] = true
				field := fmt.Sprintf("レコード種別 %s", r.Type)
				if found {
					field = fmt.Sprintf("QR #%d, レコード種別 %s", qrNum, r.Type)
				}
				results = append(results, ValidationResult{
					Level:   LevelWarning,
					Field:   field,
					Message: "文字化けが検出されました（QRコード境界での Shift-JIS 文字の分断と思われます）",
				})
				break
			}
		}
	}
	return results
}

// findQRForRecord はレコードのCSV行がどのRawQRのテキストに含まれるかを探す。
// 見つかった場合はQR番号（1始まり）とtrueを返す。
func findQRForRecord(r parser.Record, rawQRs []parser.RawQR) (int, bool) {
	line := strings.Join(r.Fields, ",")
	for i, raw := range rawQRs {
		if raw.ErrMsg != "" {
			continue
		}
		if strings.Contains(raw.Text, line) {
			return i + 1, true
		}
	}
	return 0, false
}

// checkOrphanRecords は数字以外のレコード種別（空文字を含む）を検出し WARNING を返す。
// SA QR コードが欠落すると、後続 QR の先頭が断片データとなり不正な種別が生じる。
func checkOrphanRecords(p parser.Prescription) []ValidationResult {
	var results []ValidationResult
	for _, r := range p.Records {
		if !isNumericType(r.Type) {
			results = append(results, ValidationResult{
				Level:   LevelWarning,
				Field:   "レコード種別",
				Message: fmt.Sprintf("不正なレコード種別 %q が検出されました（QRコード欠落による断片データの可能性）", r.Type),
			})
		}
	}
	return results
}

func isNumericType(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
