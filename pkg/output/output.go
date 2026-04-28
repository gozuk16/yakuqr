package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gozuk16/yakuqr/pkg/parser"
	"github.com/gozuk16/yakuqr/pkg/validator"
)

// BuildText はPrescriptionとバリデーション結果からテキストを生成する。
func BuildText(p parser.Prescription, results []validator.ValidationResult) string {
	var sb strings.Builder

	sb.WriteString("=== RAW QR DATA ===\n")
	for i, raw := range p.RawQRs {
		if raw.ErrMsg != "" {
			fmt.Fprintf(&sb, "[QR #%d] (読み取り失敗: %s)\n\n", i+1, raw.ErrMsg)
		} else {
			fmt.Fprintf(&sb, "[QR #%d]\n%s\n\n", i+1, raw.Text)
		}
	}

	sb.WriteString("=== PARSED DATA ===\n")
	fmt.Fprintf(&sb, "バージョン: %s\n", p.Version)

	if p.Version == parser.VersionUnknown {
		sb.WriteString("\n")
	} else if p.Version >= parser.Version2_1 {
		// JAHISTC04以降: レコード1=患者情報, レコード201=薬品情報
		if recs, ok := p.RecordMap["1"]; ok && len(recs) > 0 {
			sb.WriteString("--- 患者情報 ---\n")
			f := recs[0].Fields
			if len(f) > 1 {
				fmt.Fprintf(&sb, "氏名: %s\n", f[1])
			}
			if len(f) > 3 {
				fmt.Fprintf(&sb, "生年月日: %s\n", formatDate(f[3]))
			}
			if len(f) > 2 {
				fmt.Fprintf(&sb, "性別: %s\n", formatSex(f[2]))
			}
		}
		if recs, ok := p.RecordMap["201"]; ok {
			sb.WriteString("--- Rp情報 ---\n")
			for i, rec := range recs {
				f := rec.Fields
				name, dose, unit := "", "", ""
				if len(f) > 2 {
					name = f[2]
				}
				if len(f) > 3 {
					dose = f[3]
				}
				if len(f) > 4 {
					unit = f[4]
				}
				fmt.Fprintf(&sb, "Rp%d: %s %s%s\n", i+1, name, dose, unit)
			}
		}
	} else {
		// JAHISTC01〜03: レコード2=患者情報, レコード6=薬品情報
		if recs, ok := p.RecordMap["2"]; ok && len(recs) > 0 {
			sb.WriteString("--- 患者情報 ---\n")
			f := recs[0].Fields
			if len(f) > 1 {
				fmt.Fprintf(&sb, "氏名: %s\n", f[1])
			}
			if len(f) > 2 {
				fmt.Fprintf(&sb, "カナ名: %s\n", f[2])
			}
			if len(f) > 3 {
				fmt.Fprintf(&sb, "生年月日: %s\n", formatDate(f[3]))
			}
			if len(f) > 4 {
				fmt.Fprintf(&sb, "性別: %s\n", formatSex(f[4]))
			}
		}

		if recs, ok := p.RecordMap["1"]; ok && len(recs) > 0 {
			sb.WriteString("--- 処方箋情報 ---\n")
			f := recs[0].Fields
			if len(f) > 2 {
				fmt.Fprintf(&sb, "医療機関コード: %s\n", f[2])
			}
		}

		if recs, ok := p.RecordMap["6"]; ok {
			sb.WriteString("--- Rp情報 ---\n")
			for i, rec := range recs {
				f := rec.Fields
				name, dose, unit, days := "", "", "", ""
				if len(f) > 2 {
					name = f[2]
				}
				if len(f) > 3 {
					dose = f[3]
				}
				if len(f) > 4 {
					unit = f[4]
				}
				if len(f) > 6 {
					days = f[6]
				}
				fmt.Fprintf(&sb, "Rp%d: %s %s%s %s日分\n", i+1, name, dose, unit, days)
			}
		}
	}

	sb.WriteString("\n")

	sb.WriteString("=== VALIDATION RESULTS ===\n")
	if len(results) == 0 {
		sb.WriteString("問題は検出されませんでした\n")
	} else {
		for _, r := range results {
			fmt.Fprintf(&sb, "%s\n", r)
		}
	}

	return sb.String()
}

// WriteOutput はテキストをファイルに書き出し、ERROR/WARNINGをstderrにも出力する。
// srcFile を指定するとstderr出力のプレフィックスに使用する。
func WriteOutput(outPath string, p parser.Prescription, results []validator.ValidationResult, srcFile ...string) error {
	text := BuildText(p, results)
	if err := os.WriteFile(outPath, []byte(text), 0644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}

	prefix := outPath
	if len(srcFile) > 0 {
		prefix = srcFile[0]
	}
	for _, r := range results {
		if r.Level == validator.LevelError || r.Level == validator.LevelWarning {
			fmt.Fprintf(os.Stderr, "%s: %s\n", prefix, r)
		}
	}
	return nil
}

// OutputPath は入力ファイルパスから出力ファイルパスを生成する。
// outDir が空なら入力ファイルと同じディレクトリに出力する。
// existing は衝突検知用マップ（呼び出し元が管理し、本関数が更新する）。
func OutputPath(inputPath, outDir string, existing map[string]bool) string {
	// 既知の拡張子を除去してベース名を取得
	base := inputPath
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".pdf"} {
		if strings.ToLower(filepath.Ext(inputPath)) == ext {
			base = inputPath[:len(inputPath)-len(ext)]
			break
		}
	}

	candidate := base + "_out.txt"
	if outDir != "" {
		candidate = filepath.Join(outDir, filepath.Base(base)+"_out.txt")
	}

	if !existing[candidate] {
		existing[candidate] = true
		return candidate
	}

	for i := 2; ; i++ {
		stem := strings.TrimSuffix(candidate, ".txt")
		numbered := fmt.Sprintf("%s_%d.txt", stem, i)
		if !existing[numbered] {
			existing[numbered] = true
			return numbered
		}
	}
}

func formatDate(s string) string {
	if len(s) == 8 {
		return s[:4] + "-" + s[4:6] + "-" + s[6:]
	}
	return s
}

func formatSex(s string) string {
	switch s {
	case "1":
		return "男"
	case "2":
		return "女"
	default:
		return s
	}
}
