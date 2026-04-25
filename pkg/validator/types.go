package validator

import "fmt"

// Level はバリデーション結果の重大度を表す。
type Level int

const (
	LevelError   Level = iota // 仕様違反（必須フィールド欠落など）
	LevelWarning              // 推奨フィールド欠落・範囲外
	LevelInfo                 // バージョン差異などの情報
)

func (l Level) String() string {
	switch l {
	case LevelError:
		return "ERROR"
	case LevelWarning:
		return "WARNING"
	case LevelInfo:
		return "INFO"
	default:
		return "UNKNOWN"
	}
}

// ValidationResult はバリデーション1件を表す。
type ValidationResult struct {
	Level   Level
	Field   string // 対象フィールド（例: "患者氏名", "Rp1薬品コード"）
	Message string // 問題の説明
}

func (r ValidationResult) String() string {
	return fmt.Sprintf("[%s] %s: %s", r.Level, r.Field, r.Message)
}
