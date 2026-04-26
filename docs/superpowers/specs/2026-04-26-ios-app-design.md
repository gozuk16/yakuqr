# yakuqr iOSアプリ 設計ドキュメント

**日付:** 2026-04-26
**対象:** yakuqr モノリポへの iOS フロントエンド追加

---

## 概要

yakuqr の iOS フロントエンドとして、JAHIS院外処方箋QRコードをカメラでスキャンし、パース結果を表示・共有できる iPhone アプリを追加する。JAHIS パース処理は Swift で再実装する（Go コードへの依存なし）。

---

## 1. リポジトリ構成

モノリポ構成。既存の `yakuqr` リポジトリに `ios/` ディレクトリを追加する。

```
yakuqr/
├── cmd/yakuqr/              # Go CLI（既存）
├── pkg/                     # Go ライブラリ（既存）
│   ├── decoder/
│   ├── parser/
│   ├── validator/
│   └── output/
├── testdata/
│   ├── generated/           # QR画像（iOS XCTest でも共有）
│   ├── golden/
│   └── collected/
├── ios/                     # 新規追加
│   ├── yakuqr-ios.xcodeproj/
│   ├── yakuqr-ios/
│   │   ├── App/             # エントリーポイント・設定
│   │   ├── Features/
│   │   │   ├── Scanner/     # カメラ・QRスキャン画面
│   │   │   ├── Result/      # パース結果表示画面
│   │   │   └── Share/       # 共有機能
│   │   └── JAHIS/           # Swift製JAHISパーサー
│   └── yakuqr-iosTests/
│       └── JAHISTests/      # testdata/generated/ を参照
├── docs/
├── tools/
├── Makefile
├── go.mod
└── .gitignore
```

**採用理由:**
- `testdata/generated/` の QR画像を iOS の XCTest から参照でき、テストカバレッジを共有できる
- JAHIS仕様変更時に1リポジトリで追跡できる
- Go と Xcode はビルドシステムが独立しているため、混在による実害なし

---

## 2. iOSアプリ アーキテクチャ

### 2.1 画面フロー

```
ScannerView → ResultView → ShareSheet（UIActivityViewController）
```

| 画面 | 役割 |
|------|------|
| ScannerView | カメラリアルタイムスキャン（主）またはフォトライブラリ選択（副） |
| ResultView | パース結果・バリデーション結果・エラー内容を表示。共有ボタンあり |
| ShareSheet | テキストファイルとして AirDrop / Files.app / メール等へ出力 |

### 2.2 レイヤー構成（MVVM）

| レイヤー | 内容 |
|----------|------|
| View（SwiftUI） | ScannerView / ResultView — 表示のみ、ロジックなし |
| ViewModel（ObservableObject） | ScannerViewModel（スキャン状態・複数QR蓄積を管理）、ResultViewModel（パース結果・バリデーション結果を保持） |
| Domain（純粋Swift） | JAHISParser / JAHISValidator / PrescriptionTextFormatter |
| Infrastructure | AVFoundation（カメラ）、PhotosUI（フォトライブラリ）、Vision / AVMetadataObject（QR検出） |

### 2.3 分割QRの処理フロー

1. 1枚目スキャン → ScannerViewModel に蓄積（分割QRと判定）
2. 2枚目以降スキャン → 蓄積リストに追加
3. 「解析する」ボタンを常に表示し、任意のタイミングで手動パース実行
4. パース結果に未完了状態（例：「1/2枚のみ」）をエラーとして含める

---

## 3. テスト戦略

| レイヤー | テスト種別 | 備考 |
|----------|-----------|------|
| Domain（JAHISParser） | XCTest Unit | `testdata/generated/` の QR画像を共有し、Go 実装との仕様乖離を検出 |
| ViewModel | XCTest Unit | AVFoundation をモック |
| View | XCUITest | シミュレーターで実行可 |
| カメラスキャン | 手動・実機 | TestFlight 配布後に確認 |

---

## 4. エラーハンドリング

**基本方針：エラー時でも読み取れたデータとエラー内容は必ず画面に表示し、共有できる状態にする。**

### 4.1 権限エラー（カメラ・フォトライブラリ）
- システムダイアログで許可を求める
- 拒否された場合は設定アプリへのディープリンクを案内するアラートを表示

### 4.2 QR読み取りエラー
- カメラフローでは毎フレーム検出が走るためサイレントに無視してスキャン継続
- フォトライブラリ選択時にQRが見つからない場合はアラートで通知

### 4.3 JAHISパースエラー・バリデーションエラー
- パース失敗・バリデーションエラーのいずれも ResultView に遷移し、以下をすべて表示：
  - 読み取れた生のQR文字列（複数ある場合はすべて）
  - パースできた範囲の結果
  - エラー内容の一覧（エラーコード・メッセージ）
- 共有ボタンで「生QR文字列＋エラー内容」をテキストファイルとして出力可能

### 4.4 分割QRの途中終了
- タイムアウトは設けない
- 「解析する」ボタンで途中の枚数でも手動でパースを実行できる
- パース結果に「分割QR未完了（n/m枚のみ）」をエラーとして含め、共有可能

---

## 5. 配布方針

| フェーズ | 配布方法 |
|----------|---------|
| フェーズ1 | Xcode から実機に直接インストール（個人使用） |
| フェーズ2 | TestFlight（社内・特定グループ配布） |

App Store 公開は現時点でスコープ外。
