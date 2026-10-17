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

// checkSAParity は分割QR（Structured Append）の連結パリティ不一致を ERROR として返す。
// 不一致は各QRの末尾データが欠落していることを示す（発行側で誤り訂正レベルや
// 分割数を誤設定し、容量不足で切り捨てられた場合など）。セットにつき1件のみ報告する。
func checkSAParity(p parser.Prescription) []ValidationResult {
	for _, r := range p.RawQRs {
		if r.SAParityErr != "" {
			return []ValidationResult{{
				Level: LevelError,
				Field: "分割QR",
				Message: fmt.Sprintf("%s。各QRの末尾データが欠落している可能性があります"+
					"（発行側の誤り訂正レベル・分割数の設定誤りなど）", r.SAParityErr),
			}}
		}
	}
	return nil
}

// checkGarbledData はレコードフィールドに Unicode 置換文字（U+FFFD）が含まれる場合
// WARNING を返す。Shift-JIS 文字が QR 分割境界で切断されると発生する。
// 同一レコード種別かつ同一QR番号の組み合わせは1件のみ報告する。
// パリティ不一致がある場合、文字化けはデータ欠落の結果なので報告しない（checkSAParity が原因を報告する）。
func checkGarbledData(p parser.Prescription) []ValidationResult {
	if len(checkSAParity(p)) > 0 {
		return nil
	}
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
// QR境界で行が切断されている場合、再構成行の前方一致でフォールバック検索する。
func findQRForRecord(r parser.Record, rawQRs []parser.RawQR) (int, bool) {
	line := strings.Join(r.Fields, ",")
	// 完全一致検索
	for i, raw := range rawQRs {
		if raw.ErrMsg != "" {
			continue
		}
		if strings.Contains(raw.Text, line) {
			return i + 1, true
		}
	}
	// 前方一致フォールバック: QR境界で切断されたレコードを検出する。
	// 再構成行がQR内の行の続きであれば、そのQRを起源と判定する。
	typePrefix := r.Type + ","
	for i, raw := range rawQRs {
		if raw.ErrMsg != "" {
			continue
		}
		for _, rawLine := range splitOnNewlines(raw.Text) {
			if !strings.HasPrefix(rawLine, typePrefix) {
				continue
			}
			// rawLine が再構成行の先頭部分であれば、このQRが起源
			if len(rawLine) > len(typePrefix) && strings.HasPrefix(line, rawLine) {
				return i + 1, true
			}
		}
	}
	return 0, false
}

func splitOnNewlines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Split(s, "\n")
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

// checkSplit911Incomplete は RecordMap に 911 レコードが残存する場合 WARNING を返す。
// 911 レコードは通常 parse911Parts によって除去されるが、
// 分割 QR が 1 枚しかスキャンできなかった場合は除去されずに残る。
func checkSplit911Incomplete(p parser.Prescription) []ValidationResult {
	if _, ok := p.RecordMap["911"]; !ok {
		return nil
	}
	return []ValidationResult{{
		Level:   LevelWarning,
		Field:   "分割制御レコード 911",
		Message: "分割制御レコード（911）が検出されました。分割QRの一部が未取得の可能性があります",
	}}
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
