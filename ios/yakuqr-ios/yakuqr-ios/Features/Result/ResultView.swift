import SwiftUI

struct ResultView: View {
    @StateObject private var vm: ResultViewModel
    @Environment(\.dismiss) private var dismiss

    init(parseResult: ScannerViewModel.ParseResult) {
        _vm = StateObject(wrappedValue: ResultViewModel(parseResult: parseResult))
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    if vm.splitIncomplete {
                        bannerView("分割QR未完了", color: .orange)
                    }
                    if vm.hasErrors {
                        bannerView("エラーあり — 内容を確認してください", color: .red)
                    } else if vm.hasWarnings {
                        bannerView("警告あり", color: .yellow)
                    }

                    sectionHeader("読み取り結果")
                    Text("バージョン: \(vm.parseResult.prescription.version.displayName)")
                        .padding(.horizontal)

                    if !vm.parseResult.validations.isEmpty {
                        sectionHeader("バリデーション")
                        ForEach(vm.parseResult.validations, id: \.description) { v in
                            HStack(alignment: .top) {
                                Text(v.level.displayName)
                                    .font(.caption.bold())
                                    .foregroundStyle(colorFor(v.level))
                                    .frame(width: 60, alignment: .leading)
                                VStack(alignment: .leading) {
                                    Text(v.field).font(.caption.bold())
                                    Text(v.message).font(.caption)
                                }
                            }
                            .padding(.horizontal)
                        }
                    }

                    sectionHeader("生QRデータ")
                    ForEach(Array(vm.parseResult.prescription.rawQRs.enumerated()), id: \.offset) { i, raw in
                        VStack(alignment: .leading) {
                            Text("QR #\(i + 1)").font(.caption.bold())
                            Text(raw)
                                .font(.system(.caption, design: .monospaced))
                                .textSelection(.enabled)
                        }
                        .padding(.horizontal)
                    }
                }
                .padding(.vertical)
            }
            .navigationTitle("解析結果")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("閉じる") { dismiss() }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button("共有") { vm.showShareSheet = true }
                        .sheet(isPresented: $vm.showShareSheet) {
                            ShareSheet(
                                text: vm.shareText,
                                filename: vm.shareFilename
                            )
                        }
                }
            }
        }
    }

    private func bannerView(_ text: String, color: Color) -> some View {
        Text(text)
            .font(.subheadline.bold())
            .frame(maxWidth: .infinity)
            .padding(.vertical, 8)
            .background(color.opacity(0.2))
            .padding(.horizontal)
    }

    private func sectionHeader(_ title: String) -> some View {
        Text(title)
            .font(.headline)
            .padding(.horizontal)
            .padding(.top, 8)
    }

    private func colorFor(_ level: ValidationLevel) -> Color {
        switch level {
        case .error: return .red
        case .warning: return .orange
        case .info: return .blue
        }
    }
}
