# yakuqr

JAHIS院外処方箋２次元シンボル記録条件規約（Ver.2〜Ver.4）に準拠したQRコードを画像・PDFファイルから読み取り、解析結果とバリデーション結果をテキストファイルに出力するCLIツール。

## インストール

```bash
go install github.com/gozuk16/yakuqr/cmd/yakuqr@latest
```

またはソースからビルド:

```bash
git clone https://github.com/gozuk16/yakuqr.git
cd yakuqr
make build
```

## 使い方

```bash
# 単一ファイル（画像またはPDF）
yakuqr prescription.pdf          # → prescription_out.txt

# 出力先を指定
yakuqr -o result.txt scan.png    # → result.txt

# 複数ファイル
yakuqr img1.png img2.png         # → img1_out.txt, img2_out.txt

# 複数ファイル + 出力ディレクトリ指定
yakuqr -d ./output img1.png img2.png

# ヘルプ
yakuqr --help
```

## 出力形式

入力ファイル名に `_out.txt` を付加したファイルに以下のセクションを出力します。

```
=== RAW QR DATA ===
[QR #1]
<生のQRコードデータ（UTF-8変換済み）>

=== PARSED DATA ===
バージョン: Ver.4
--- 患者情報 ---
氏名: 山田 太郎
...

=== VALIDATION RESULTS ===
[ERROR] 患者氏名: 患者氏名（レコード2 フィールド2）が空です
[WARNING] 薬品情報(レコード6): レコード種別6（処方薬品情報）が存在しません
```

ERROR/WARNINGレベルの問題は標準エラー出力にも表示されます。

## 終了コード

| コード | 意味 |
|--------|------|
| `0` | 正常終了 |
| `1` | QR読み取り/IO失敗 |
| `2` | ERRORバリデーションあり |
| `64` | 引数不正 |

## 対応ファイル形式

| 形式 | 説明 |
|------|------|
| PNG / JPEG | スキャンまたはキャプチャした処方箋画像 |
| PDF | 埋め込み画像としてQRコードが含まれるPDF |

> **注意:** PDFはページ内に埋め込まれた画像からQRを検出します。ベクター描画されたQRコードは対象外です。

## ビルド・開発

```bash
make build   # バイナリをビルド
make test    # 全テストを実行
make lint    # golangci-lintを実行
make clean   # ビルド成果物を削除
```
