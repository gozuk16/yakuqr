# TODO

## 分割QR（Structured Append）末尾切り捨て不具合のハンドリング検証

背景: 他社ベンダーの薬情出力アプリが誤り訂正レベルと分割数を誤設定し、各QRの末尾データが容量不足で切り捨てられた。
yakuqr はこれを「Shift-JIS 文字が分割境界で分断された」と誤認した。正しく見分けられるか、意図的に壊したテストQRで検証する。

### Phase 1: 再現（根本原因調査）
- [x] SA分割QRを任意のバイト列・ヘッダで生成できるテストデータ生成ツールを作成（`tools/gen-sa-testdata-qr`, `make gen-sa-testdata`）
- [x] テストQRを作成（`testdata/generated/sa/`）
  - [x] A: 正常な分割QR（2バイト文字が境界をまたぐ・欠落なし）
  - [x] B: 末尾切り捨て（切断点が2バイト文字の途中）
  - [x] C: 末尾切り捨て（切断点が文字境界＝文字化けなしの静かな欠落）
  - [x] D: 文字数指示は元の長さのまま、ビット列だけ途中で切れている
  - [x] E: 事例の再現（誤り訂正L前提で型番を決め、Mで出力して末尾が切れる）
- [x] 各データを yakuqr にかけ、現状の出力を記録する

#### 結果（現状の yakuqr）
| データ | 期待 | 現状 |
|---|---|---|
| A 正常 | 警告なし | ✗ 全体が文字化けし「Shift-JIS分断」警告5件（誤検知） |
| B 途中切れ | 欠落を検出 | ✗ 「Shift-JIS分断」と誤認 |
| C 境界切れ | 欠落を検出 | ✗ 「問題なし」（住所が「東京都c区」に化けたまま素通り） |
| D 文字数超過 | 読み取り失敗を報告 | △ 読み取り失敗は報告されるが、バージョン検出ERRORも出る |
| E 事例再現 | 欠落を検出 | ✗ 「Shift-JIS分断」と誤認（事例と同じ症状） |

#### 根本原因
1. `pkg/decoder`: SA の各シンボルを gozxing の `GetText()` で1枚ずつ文字列化しており、文字コードもシンボルごとに推定される。
   2バイト文字が境界をまたぐと推定が外れ、シンボル全体が化ける（規格上はバイト列を連結してから文字コード変換すべき）。
2. SA ヘッダの連結パリティ（全データのXOR）を検証していないため、末尾切り捨てを検出できない。

#### 仮説検証（scratch）
バイト連結→Shift_JIS変換＋パリティ照合で A は完全復元・パリティ一致、B/C/E はパリティ不一致で全件検出できることを確認。

### Phase 2〜4
- [ ] 修正方針をユーザーと相談（decoder の SA 処理変更・パリティ検証、iOS版への同期要否）

## GoReleaser + GitHub Actions によるMac/Windowsバイナリ配布セットアップ

- [x] `main.go` にldflagsでバージョンを注入できる仕組みを追加
- [x] `.goreleaser.yaml` を作成（darwin amd64/arm64, windows amd64向けビルド設定）
- [x] `.github/workflows/release.yml` を作成（`v*`タグpushでGoReleaser実行）
- [x] Makefileに `release-snapshot`（ローカル動作確認用）ターゲットを追加
- [x] ローカルで `goreleaser release --snapshot --clean` を実行し動作確認（darwin amd64/arm64, windows amd64のビルド成功）
- [x] README.mdにダウンロード手順・Mac Gatekeeper対応を追記
- [x] CHANGELOG.mdに変更履歴を追記
- [x] プルリクエスト作成

### 補足
- 実際のリリースは `git tag v0.2.0 && git push origin v0.2.0` のように `v*` タグをpushすると自動実行される（このPRではワークフロー追加のみで、タグはまだ作成していない）
