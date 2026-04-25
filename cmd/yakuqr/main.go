package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/gozuk16/yakuqr/pkg/decoder"
	"github.com/gozuk16/yakuqr/pkg/output"
	"github.com/gozuk16/yakuqr/pkg/parser"
	"github.com/gozuk16/yakuqr/pkg/validator"
)

func main() {
	app := &cli.App{
		Name:    "yakuqr",
		Usage:   "JAHIS院外処方箋QRコードを読み取り、内容を解析してテキストファイルに出力します",
		Version: "0.1.0",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "出力ファイルパス（単一ファイル入力時のみ有効）",
			},
			&cli.StringFlag{
				Name:    "output-dir",
				Aliases: []string{"d"},
				Usage:   "出力ディレクトリ（複数ファイル入力時に使用）",
			},
		},
		Action: run,
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(64)
	}
}

func run(c *cli.Context) error {
	files := c.Args().Slice()
	if len(files) == 0 {
		return cli.ShowAppHelp(c)
	}

	outFlag := c.String("output")
	outDir := c.String("output-dir")

	if outFlag != "" && len(files) > 1 {
		return cli.Exit("複数ファイル指定時に -o は使用できません。--output-dir を使用してください", 64)
	}

	existing := make(map[string]bool)
	successCount, failCount, errorCount := 0, 0, 0

	for _, inputPath := range files {
		rawQRs, decErrs := decoder.DecodeFile(inputPath)
		if len(decErrs) > 0 && len(rawQRs) == 0 {
			fmt.Fprintf(os.Stderr, "%s: QR読み取り失敗: %v\n", inputPath, decErrs[0])
			failCount++
			continue
		}

		p, msgs := parser.Parse(rawQRs)
		for _, msg := range msgs {
			fmt.Fprintf(os.Stderr, "%s: %s\n", inputPath, msg)
		}

		results := validator.Validate(p)

		var outPath string
		if outFlag != "" {
			outPath = outFlag
		} else {
			outPath = output.OutputPath(inputPath, outDir, existing)
		}

		if err := output.WriteOutput(outPath, p, results, inputPath); err != nil {
			fmt.Fprintf(os.Stderr, "%s: 出力ファイル書き込み失敗: %v\n", inputPath, err)
			failCount++
			continue
		}

		fmt.Printf("%s -> %s\n", inputPath, outPath)
		successCount++

		for _, r := range results {
			if r.Level == validator.LevelError {
				errorCount++
			}
		}
	}

	if len(files) > 1 {
		fmt.Printf("\n処理完了: 成功 %d件 / 失敗 %d件 / ERRORバリデーション %d件\n", successCount, failCount, errorCount)
	}

	if failCount > 0 {
		os.Exit(1)
	}
	if errorCount > 0 {
		os.Exit(2)
	}
	return nil
}
