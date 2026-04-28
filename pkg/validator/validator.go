package validator

import "github.com/gozuk16/yakuqr/pkg/parser"

// Validate はPrescriptionをJAHIS規約に照らして検証し、結果リストを返す。
func Validate(p parser.Prescription) []ValidationResult {
	// QR読み取り品質の共通チェック（全バージョン）
	results := checkQRReadFailures(p)
	results = append(results, checkGarbledData(p)...)
	results = append(results, checkOrphanRecords(p)...)

	// バージョン固有チェック
	for _, r := range rulesFor(p.Version) {
		ok, msg := r.check(p)
		if !ok {
			results = append(results, ValidationResult{
				Level:   r.level,
				Field:   r.field,
				Message: msg,
			})
		}
	}
	return results
}
