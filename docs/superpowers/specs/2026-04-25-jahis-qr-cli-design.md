# JAHIS院外処方箋QRコード読み取りCLI 設計書

**日付:** 2026-04-25  
**プロジェクト:** yakuqr  
**言語:** Go

---

## 概要

JAHISの院外処方箋２次元シンボル記録条件規約（Ver.2〜Ver.4系）に準拠したQRコードを読み取るCLIツール。画像ファイル（PNG/JPEG）またはPDFファイルを入力として受け取り、QRコードの内容を解析してテキストファイルに出力する。また、JAHIS仕様との適合チェックを行い、不適合項目を指摘する。

---

## ディレクトリ構成

```
yakuqr/
├── cmd/yakuqr/
│   └── main.go          ← CLIエントリポイント（urfave/cli）
├── pkg/
│   ├── decoder/         ← QRコード読み取り（画像・PDF対応）
│   ├── parser/          ← JAHISデータ解析（バージョン自動検出）
│   ├── validator/       ← JAHIS仕様検証
│   └── output/          ← テキスト出力フォーマット
├── testdata/            ← テスト用サンプルファイル
├── Makefile
├── go.mod
├── CHANGELOG.md
└── README.md
```

---

## 主要外部ライブラリ

| ライブラリ | 用途 |
|------------|------|
| `github.com/urfave/cli/v2` | CLIフレームワーク |
| `github.com/makiuchi-d/gozxing` | Pure GoのQRデコーダー |
| `github.com/pdfcpu/pdfcpu` | PDFから画像ページを抽出 |

---

## CLIインターフェース

```bash
yakuqr [options] <file1> [file2 ...]

# 例
yakuqr prescription.pdf          # → prescription_out.txt
yakuqr -o result.txt scan.png    # → result.txt
yakuqr img1.png img2.png         # 複数ファイル処理
```

**オプション:**
- `-o, --output <file>` — 出力ファイルパスを指定（単一ファイル入力時のみ有効）
- `--help` — ヘルプを表示
- `--version` — バージョンを表示

---

## データフロー

```
入力ファイル（PNG/JPEG/PDF）
    ↓
[decoder] ファイルタイプ判別
    ↓
    ├─ 画像: gozxingでQRコードを検出・デコード（複数QR対応）
    └─ PDF: pdfcpuでページを画像に変換 → gozxingでデコード
    ↓
生のQRデータ文字列リスト（Shift_JIS→UTF-8変換済み）
    ↓
[parser] 分割QRコードの検出・結合（JAHISの分割符号に基づく）
    ↓
バージョン自動検出（ヘッダフィールドから判別）
    ↓
構造化された処方箋データ（フィールドIDと値のマップ）
    ↓
[validator] JAHISルール検証
    ↓
バリデーション結果リスト（ERROR/WARNING/INFO）
    ↓
[output] テキストファイル出力          stderr出力
  ・生のQRデータ                      ・ERROR/WARNING を表示
  ・解析済み処方箋フィールド
  ・バリデーション結果
```

---

## パッケージ詳細

### pkg/decoder

- 入力ファイルの拡張子・MIMEタイプでPDF／画像を判別
- 画像ファイル: `gozxing` でQRコードを全件検出・デコード
- PDFファイル: `pdfcpu` で各ページを画像に変換後、`gozxing` でデコード
- Shift_JIS → UTF-8 変換を実施
- 戻り値: `[]string`（生QRデータのリスト）

### pkg/parser

- JAHISの分割符号（先頭レコードの識別子）を検出し、複数QRを結合
- ヘッダレコードからバージョン（Ver.2〜Ver.4）を自動検出
- カンマ区切りのレコードを解析し、フィールドIDと値にマッピング

**主要フィールド識別子:**

| 識別子 | 内容 |
|--------|------|
| `1` | 患者情報（氏名、カナ名、生年月日、性別など） |
| `2` | 医療機関情報（コード、名称など） |
| `3` | 処方医情報 |
| `5` | Rp情報（薬品コード、薬品名、用量、単位、用法など） |

- 戻り値: `Prescription` 構造体（バージョン情報・フィールドマップ・生データ含む）

### pkg/validator

- バージョン別のルールセットを保持
- 検出バージョンに応じたルールを適用して検証

**バリデーションレベル:**

| レベル | 例 |
|--------|----|
| `ERROR` | 必須フィールドの欠落、フォーマット不正（生年月日形式など） |
| `WARNING` | 推奨フィールドの欠落、値が仕様範囲外 |
| `INFO` | バージョン間の仕様差異による注意事項 |

- 戻り値: `[]ValidationResult`

### pkg/output

- `Prescription` と `[]ValidationResult` を受け取りテキストを生成
- 出力ファイルへの書き込み
- ERROR/WARNING を stderr に出力

---

## 出力ファイルフォーマット

```
=== RAW QR DATA ===
[QR #1]
<生のQRコードデータ（UTF-8変換済み）>

[QR #2]
<生のQRコードデータ>

=== PARSED DATA ===
バージョン: Ver.4
--- 患者情報 ---
氏名: 山田 太郎
カナ名: ヤマダ タロウ
生年月日: 1970-01-01
性別: 男
--- 医療機関情報 ---
...
--- Rp情報 ---
Rp1: アムロジピン錠5mg  1錠  1日1回朝食後  28日分
...

=== VALIDATION RESULTS ===
[ERROR] 患者ID: 必須フィールドが欠落しています
[WARNING] 薬品コード Rp1: HOTコードが空です（推奨フィールド）
[INFO] Ver.3形式のフィールドを検出しました
```

---

## エラーハンドリング方針

- QR読み取り失敗 → stderrにエラー表示してそのファイルをスキップ、次ファイルへ継続
- 複数ファイル指定時は全ファイルを処理し、最後に処理サマリを表示
- 出力ファイル名のデフォルト: 入力ファイル名 + `_out.txt`

---

## テスト方針

- `pkg/parser` と `pkg/validator` はテーブルドリブンテストで各JAHISルールを検証
- `testdata/` にサンプルQRテキストを置いてパーサーの単体テストを実施
- `go test ./...` で全テスト実行（Makefileに `make test` として定義）

---

## Makefile ターゲット

| ターゲット | 内容 |
|------------|------|
| `make build` | バイナリをビルド |
| `make test` | 全テストを実行 |
| `make lint` | golangci-lintを実行 |
| `make clean` | ビルド成果物を削除 |
