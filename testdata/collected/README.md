# testdata/collected/

このディレクトリには、インターネットから収集した実世界の JAHIS 処方箋 QR コードサンプルを格納します。

**著作権の関係上、画像ファイルはリポジトリにコミットしません。**
テストを実行する前に、以下の手順で手動ダウンロードして配置してください。

---

## ITmedia サンプル画像（架空データ使用）

出典記事: https://nlab.itmedia.co.jp/cont/articles/3293608/

著作権: Copyright © ITmedia Inc. All Rights Reserved.
利用条件: テスト目的の個人利用のみ。再配布・リポジトリへのコミット不可。

### ダウンロード手順

```bash
cd testdata/collected/

curl -O "https://preresearch.image.itmedia.co.jp/nl/articles/1907/25/miya_1907okusuriqr01.jpg"
curl -O "https://preresearch.image.itmedia.co.jp/nl/articles/1907/25/miya_1907okusuriqr02.jpg"
curl -O "https://preresearch.image.itmedia.co.jp/nl/articles/1907/25/miya_1907okusuriqr03.jpg"
```

ダウンロード後、`go test ./pkg/decoder/ -run TestDecodeImage_Collected` でテストが実行されます。

---

## サンプルを追加する場合

1. 著作権・利用条件を確認し、テスト目的での利用が許可されているもののみ追加してください
2. このREADMEにファイル名・出典URL・著作権情報・取得日を記載してください
3. 画像ファイルは `.gitignore` 対象のため、コミットしないでください
