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
yakuqr img1.png img2.png         # → img1_out.txt, img2_out.txt
```

**オプション:**
- `-o, --output <file>` — 出力ファイルパスを指定。複数ファイル入力時に指定された場合はエラーとし、`--output-dir` の使用を促す
- `-d, --output-dir <dir>` — 出力ディレクトリを指定。各入力ファイルに対し `<dir>/<入力ファイル名>_out.txt` を生成。未指定時は入力ファイルと同じディレクトリに出力
- `--help` — ヘルプを表示
- `--version` — バージョンを表示

**終了コード:**
- `0` — 全ファイルが正常に処理され、ERRORレベルのバリデーションがゼロ
- `1` — いずれかのファイルでQR読み取り／パース／IOに失敗
- `2` — 全ファイル処理は完了したが、ERRORレベルのバリデーションが1件以上発生
- `64` — CLI引数が不正（urfave/cli の標準的な扱いに準拠）

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
  ・生のQRデータ                      ・ERROR/WARNING のみ表示
  ・解析済み処方箋フィールド            （INFOは出さない）
  ・バリデーション結果（ERROR/WARNING/INFO 全て）
```

---

## パッケージ詳細

### pkg/decoder

- 入力ファイルの拡張子（`.png`/`.jpg`/`.jpeg`/`.pdf`）とマジックバイトでPDF／画像を判別。両者が一致しない場合はマジックバイト優先
- 画像ファイル: `gozxing` で画像内のQRコードを全件検出・デコード（複数QRの並びは画像座標の上→下、左→右でソート）
- PDFファイル: `pdfcpu` で各ページをPNGラスタ画像に変換し、ページ順に `gozxing` でデコード。複数ページ間でも検出順は維持
- JAHIS規約上、QRコードのデータはShift_JISで符号化される前提。`gozxing` から取得したバイト列を Shift_JIS → UTF-8 へ変換した結果を返す。変換に失敗したQRは ERROR を付与してそのQRを除外
- 戻り値: `([]string, []DecodeError)` — 生QRデータのリストと、デコード／変換失敗の詳細

### pkg/parser

- JAHIS規約の分割QR表現（先頭レコードに分割番号Nと分割総数Mが含まれる形式）を検出し、N順に結合する
  - 具体的なレコード番号・フィールド位置は実装着手時にJAHIS規約Ver.2〜Ver.4の最新版を参照して確定する。本設計では「分割番号Nと総数Mが取得できる前提のロジック」までを扱う
  - 分割総数Mが1のQRが1件のみ、または分割符号が無いQRはそのまま単独処方箋として扱う
  - 同一画像/PDF内に分割QRと非分割QRが混在した場合は、分割番号で連結できたものを1処方箋、残りを独立した処方箋として個別に処理
- 分割QRが揃っていない場合（一部のNが欠落）は WARNING を出力し、取得済み分のみで処理を続行
- ヘッダレコードからバージョン（Ver.2／Ver.3／Ver.4）を自動検出。検出不能な場合は Ver.4（最新版）として扱い INFO を出力
- カンマ区切り（CSV相当）のレコードを解析し、レコード番号（先頭フィールド）をキーとして値にマッピング

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
- 出力ファイルへの書き込み（UTF-8、改行コードはLF）
- ERROR/WARNING のみを stderr に出力（INFOはstderrには出さず、出力ファイルにのみ記録）
- stderr 出力フォーマット例: `<入力ファイル名>: [ERROR] <メッセージ>`

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
- パース失敗（JAHIS規約に致命的に反する）→ そのファイルの出力ファイルにバリデーション結果のみを書き、次ファイルへ継続
- 複数ファイル指定時は全ファイルを処理し、最後に処理サマリ（成功件数／失敗件数／ERROR件数）を表示
- 出力ファイル名のデフォルト: 入力ファイルの拡張子を除いた名前 + `_out.txt`（例: `scan.png` → `scan_out.txt`、`prescription.pdf` → `prescription_out.txt`）
- 入力ファイル名に同名の拡張子違いが混在しても出力ファイルが衝突しないよう、衝突検知時はサフィックス `_out_2.txt`, `_out_3.txt` を採番

---

## テスト方針

- `pkg/parser` と `pkg/validator` はテーブルドリブンテストで各JAHISルール（Ver.2〜Ver.4）を検証
- `pkg/decoder` は `testdata/` 配下のサンプル画像・PDFを用いた統合的な単体テストを実施（gozxing/pdfcpuの呼び出し含む）
- `pkg/output` はゴールデンファイル方式（期待出力テキストとの比較）で検証
- `testdata/` には以下を配置:
  - JAHIS Ver.2／Ver.3／Ver.4 各バージョンのサンプルQRテキスト
  - 単一QR画像、複数QR画像、分割QR画像、PDF（単一ページ・複数ページ）
- `go test ./...` で全テスト実行（Makefileに `make test` として定義）

---

## Makefile ターゲット

| ターゲット | 内容 |
|------------|------|
| `make build` | バイナリをビルド |
| `make test` | 全テストを実行 |
| `make lint` | golangci-lintを実行 |
| `make clean` | ビルド成果物を削除 |
