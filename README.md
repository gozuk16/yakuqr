# yakuqr

JAHIS院外処方箋２次元シンボル記録条件規約（Ver.1.0〜Ver.2.6）に準拠した処方箋QRコードを解析・バリデーションするツール群。

- **CLIツール（Go）** — 画像・PDFファイルを入力し、解析結果とバリデーション結果をテキストファイルへ出力
- **iOSアプリ（SwiftUI）** — カメラまたはフォトライブラリからQRコードをスキャンし、解析結果を画面表示・共有

---

## CLIツール

### インストール

```bash
go install github.com/gozuk16/yakuqr/cmd/yakuqr@latest
```

またはソースからビルド:

```bash
git clone https://github.com/gozuk16/yakuqr.git
cd yakuqr
make build
```

### 使い方

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

### 出力形式

入力ファイル名に `_out.txt` を付加したファイルに以下のセクションを出力します。

```
=== RAW QR DATA ===
[QR #1]
<生のQRコードデータ（UTF-8変換済み）>

=== PARSED DATA ===
バージョン: Ver.2.6
--- 患者情報 ---
氏名: 山田 太郎
...

=== VALIDATION RESULTS ===
[ERROR] 患者氏名: 患者氏名（レコード1 フィールド2）が空です
[WARNING] 薬品情報(レコード201以上): レコード種別201以上（薬品情報）が存在しません
```

ERROR/WARNINGレベルの問題は標準エラー出力にも表示されます。

### 終了コード

| コード | 意味 |
|--------|------|
| `0` | 正常終了 |
| `1` | QR読み取り/IO失敗 |
| `2` | ERRORバリデーションあり |
| `64` | 引数不正 |

### 対応ファイル形式

| 形式 | 説明 |
|------|------|
| PNG / JPEG | スキャンまたはキャプチャした処方箋画像 |
| PDF | 埋め込み画像としてQRコードが含まれるPDF |

> **注意:** PDFはページ内に埋め込まれた画像からQRを検出します。ベクター描画されたQRコードは対象外です。

### ビルド・開発

```bash
make build   # バイナリをビルド
make test    # 全テストを実行
make lint    # golangci-lintを実行
make clean   # ビルド成果物を削除
```

---

## iOSアプリ

### 概要

SwiftUIで実装されたiOSアプリ。iPhone/iPadのカメラで処方箋QRコードをスキャンし、JAHIS規約に基づく解析・バリデーション結果を表示します。

- **動作環境:** iOS 16.6 以降
- **言語:** Swift 5（SwiftUI）

### 主な機能

| 機能 | 説明 |
|------|------|
| カメラスキャン | リアルタイムでQRコードを読み取り、複数枚を連続スキャン |
| 画像読み込み | フォトライブラリから処方箋画像を選択してQRを抽出 |
| 分割QR対応 | JAHISTC形式の分割QRコードを自動集積・結合 |
| 解析表示 | バージョン情報・バリデーション結果・生QRデータを画面表示 |
| 結果共有 | 解析結果テキストをファイルとして他のアプリへ共有 |

### 対応バージョン

| バージョン | 説明 |
|-----------|------|
| Ver.1.0 / Ver.1.1 | 旧形式（レコード2で患者情報、レコード6で薬品情報） |
| Ver.2.0 | 旧形式（同上） |
| Ver.2.1 | 新形式（レコード1で患者情報、レコード201以上で薬品情報） |
| Ver.2.6 | 新形式（JAHISTC分割ヘッダによるバージョン検出） |

### バリデーション

| レベル | 内容 |
|--------|------|
| ERROR | 患者氏名・生年月日の欠損など、必須フィールドの不備 |
| WARNING | 薬品情報レコードの欠損、QR読み取り失敗、文字化け、不正レコード種別、分割QR未完了 |
| INFO | Ver.1.x/2.0 形式の互換性通知 |

### ビルド

Xcodeでビルドします。App Storeへの配布は未対応のため、実機への転送にはApple Developerアカウントが必要です。

```
ios/yakuqr-ios/yakuqr-ios.xcodeproj
```

1. `ios/yakuqr-ios/yakuqr-ios.xcodeproj` をXcodeで開く
2. Signing & Capabilities でチームを設定
3. ビルド・実機転送
