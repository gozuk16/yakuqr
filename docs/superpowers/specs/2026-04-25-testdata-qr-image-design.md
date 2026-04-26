# テストデータ QR画像管理 設計仕様

## 目的

デコーダーの統合テスト（画像ファイル → QRデコード → テキスト化）とバリデーションの網羅テストを確立する。テストデータは「自動生成」と「インターネット収集」に分けてディレクトリで管理する。

## ディレクトリ構成

```
testdata/
  *.txt                    既存（パーサー・バリデーターテスト用テキスト）
  golden/                  既存（ゴールデンファイル）
  generated/               go generate で生成、リポジトリにコミット
    qr_ver2_single.png
    qr_ver3_single.png
    qr_ver4_single.png
    qr_ver4_split_1.png
    qr_ver4_split_2.png
    README.md              再生成手順
  collected/               .gitignore 対象、手動配置のみ
    README.md              ITmedia サンプル画像のDL手順・URL記載
    .gitkeep

tools/
  gen-testdata-qr/
    main.go                QR画像生成ツール
```

## QR画像生成ツール

### 仕様

- パス: `tools/gen-testdata-qr/main.go`
- ライブラリ: `github.com/skip2/go-qrcode`
- 入力: `testdata/*.txt` の各ファイル
- 出力: `testdata/generated/qr_{stem}.png`（stem = 拡張子を除いたファイル名）
- QRコードサイズ: 256x256 px、誤り訂正レベル M
- 起動方法:
  - `go run ./tools/gen-testdata-qr/`
  - `make gen-testdata`

### 起動方法

`make gen-testdata` のみとする（`go generate` は使わない）。生成は CI では実行せず、開発者が手動で実行してコミットする。

## テスト設計

### 1. デコーダー統合テスト（generated/、常時実行）

`pkg/decoder/decoder_test.go` にテーブル駆動テストを追加:

```go
var decoderImageCases = []struct {
    file       string
    wantSubstr string
}{
    {"qr_ver4_single.png",  "JAHISTC04"},
    {"qr_ver4_split_1.png", "JAHISTC04,1"},
    {"qr_ver4_split_2.png", "JAHISTC04,2"},
    {"qr_ver2_single.png",  "1,2,"},
    {"qr_ver3_single.png",  "1,3,"},
}

func TestDecodeImage_Generated(t *testing.T) {
    for _, tc := range decoderImageCases {
        tc := tc
        t.Run(tc.file, func(t *testing.T) {
            path := testdataPath(filepath.Join("generated", tc.file))
            results, errs := decoder.DecodeFile(path)
            if len(errs) > 0 {
                t.Fatalf("decode errors: %v", errs)
            }
            if len(results) == 0 {
                t.Fatal("no QR decoded")
            }
            if !strings.Contains(results[0], tc.wantSubstr) {
                t.Errorf("expected %q in result, got: %q", tc.wantSubstr, results[0])
            }
        })
    }
}
```

### 2. 実世界 E2E テスト（collected/、ファイルがあれば実行）

```go
func TestDecodeImage_Collected(t *testing.T) {
    dir := testdataPath("collected")
    files, _ := filepath.Glob(filepath.Join(dir, "*.jpg"))
    files2, _ := filepath.Glob(filepath.Join(dir, "*.png"))
    files = append(files, files2...)
    if len(files) == 0 {
        t.Skip("no collected testdata; see testdata/collected/README.md")
    }
    for _, f := range files {
        f := f
        t.Run(filepath.Base(f), func(t *testing.T) {
            results, errs := decoder.DecodeFile(f)
            if len(errs) > 0 {
                t.Logf("decode errors (non-fatal): %v", errs)
            }
            if len(results) == 0 {
                t.Fatal("no QR decoded")
            }
        })
    }
}
```

### 3. E2E パイプラインテスト（generated/、新規ファイル）

`pkg/parser/integration_test.go` を新設（`package parser_test`）。`decoder` と `parser` を両方インポートし、画像 → decode → parse → validate の全パイプラインを検証する。

```go
var pipelineCases = []struct {
    imageFile   string
    wantVersion parser.Version
    wantRecord  string // RecordMap に存在すべきキー
}{
    {"qr_ver4_single.png",  parser.Version4, "1"},
    {"qr_ver2_single.png",  parser.Version2, "2"},
    {"qr_ver3_single.png",  parser.Version3, "2"},
}

func TestPipeline_DecodeParseValidate(t *testing.T) { ... }

func TestPipeline_SplitQR_Combined(t *testing.T) {
    // qr_ver4_split_1.png + qr_ver4_split_2.png を両方デコードして Parse に渡す
    // → RecordMap["201"] が存在し、バリデーションエラーがないことを確認
}
```

## collected/README.md の内容

以下の情報を記載:

1. ITmedia 記事のサンプル画像（架空データ使用、著作権: ITmedia Inc.）
   - URL: https://preresearch.image.itmedia.co.jp/nl/articles/1907/25/miya_1907okusuriqr01.jpg
   - URL: https://preresearch.image.itmedia.co.jp/nl/articles/1907/25/miya_1907okusuriqr02.jpg
   - URL: https://preresearch.image.itmedia.co.jp/nl/articles/1907/25/miya_1907okusuriqr03.jpg
   - 出典記事: https://nlab.itmedia.co.jp/cont/articles/3293608/
   - 手動ダウンロード手順（curl コマンド例）

2. ダウンロード後のファイル配置先: `testdata/collected/`

3. 著作権注記: 著作権はITmedia Inc.に帰属。テスト目的の個人利用のみ。リポジトリへのコミット不可。

## .gitignore 追加

```
testdata/collected/*.jpg
testdata/collected/*.png
testdata/collected/*.pdf
```

## Makefile ターゲット追加

```makefile
gen-testdata:  ## testdata/generated/ の QR画像を再生成
	go run ./tools/gen-testdata-qr/
```

## 実装順序

1. `github.com/skip2/go-qrcode` を go.mod に追加
2. `tools/gen-testdata-qr/main.go` を実装
3. `make gen-testdata` で `testdata/generated/*.png` を生成
4. `pkg/decoder/decoder_test.go` にテーブル駆動テストを追加
5. `pkg/parser/integration_test.go` を新設
6. `testdata/collected/README.md` と `.gitkeep` を作成
7. `.gitignore` を更新
8. `Makefile` を更新
