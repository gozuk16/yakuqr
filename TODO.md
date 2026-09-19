# TODO

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
