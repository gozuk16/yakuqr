package validator

import "github.com/gozuk16/yakuqr/pkg/parser"

// Validate はPrescriptionをJAHIS規約に照らして検証し、結果リストを返す。
func Validate(p parser.Prescription) []ValidationResult {
	rules := rulesFor(p.Version)
	var results []ValidationResult
	for _, r := range rules {
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
