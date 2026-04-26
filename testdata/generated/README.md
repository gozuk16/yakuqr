# testdata/generated/

このディレクトリには `testdata/*.txt` から自動生成した JAHIS 形式 QR コード画像を格納しています。

## 再生成手順

リポジトリルートで以下を実行してください:

```bash
make gen-testdata
# または
go run ./tools/gen-testdata-qr/
```

## ファイル一覧

| ファイル | 元データ | 説明 |
|---------|---------|------|
| qr_ver2_single.png | ver2_single.txt | JAHIS Ver.2 単一QR |
| qr_ver3_single.png | ver3_single.txt | JAHIS Ver.3 単一QR |
| qr_ver4_single.png | ver4_single.txt | JAHIS Ver.4 単一QR（JAHISTC04形式） |
| qr_ver4_split_1.png | ver4_split_1.txt | JAHIS Ver.4 分割QR その1 |
| qr_ver4_split_2.png | ver4_split_2.txt | JAHIS Ver.4 分割QR その2 |
| qr_ver4_split_only2.png | ver4_split_only2.txt | 分割QR欠落テスト用（その2のみ） |

## 注意

- エンコーディングは Shift_JIS（実際の JAHIS QR コードに準拠）
- サイズ: 256×256 px、誤り訂正レベル M
- これらのファイルはリポジトリにコミットされます
