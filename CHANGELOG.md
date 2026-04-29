# Changelog

## [0.1.3] - 2026-04-29
### Added
- JAHISTC連番3分割・911累積型3分割のテストデータ（6ファイル）と QR 画像を追加
- バリデーター: 911分割制御レコードが残存する場合（分割QRが未取得）に WARNING を出力
- 統合テスト6ケース追加（3分割全枚・QR欠落・部分スキャンの各シナリオ）
### Changed
- QR画像生成サイズを 256px → 512px に変更し、大容量テストデータの確実なデコードに対応

## [0.1.2] - 2026-04-26
### Added
- `testdata/generated/` に QR 画像テストデータ（go-qrcode で自動生成）を追加
- `pkg/decoder/decoder_test.go` にテーブル駆動デコーダー統合テストを追加
- `pkg/parser/integration_test.go` に decoder→parser→validator E2E パイプラインテストを追加
- `testdata/collected/README.md` に ITmedia サンプルの手動ダウンロード手順を記載
- `make gen-testdata` で QR 画像を再生成できる Makefile ターゲットを追加

## [0.1.1] - 2026-04-25
### Fixed
- gozxing はQRコードをUTF-8で返すため、`toUTF8()`による二重変換を除去し文字化けを修正
- 分割QRのヘッダ検出を `fields[0]=="9"` から実際のJAHIS形式 `JAHISTC{ver},{seq}` に修正
- Ver.4 のバージョン検出を JAHISTC ヘッダから行うように修正
- Ver.4 のバリデーションルールを実際のレコード種別（患者情報→レコード1、薬品情報→レコード201以上）に対応
- テストデータ（ver4_*.txt）を実際のJAHIS Ver.4 形式に更新

## [0.1.0] - 2026-04-25
### Added
- JAHIS院外処方箋QRコード（Ver.2〜Ver.4）の読み取り対応
- 画像ファイル（PNG/JPEG）・PDFファイルの入力対応
- 分割QRコードの自動結合（JAHIS規約の分割符号に基づく）
- JAHIS仕様バリデーション（ERROR/WARNING/INFO）
- テキストファイルへの出力（生データ・解析結果・バリデーション結果）
- ERROR/WARNINGを標準エラー出力にも表示
- `-o` / `--output-dir` による出力先指定
- 出力ファイル名衝突時のサフィックス採番
- 終了コード（0/1/2/64）
