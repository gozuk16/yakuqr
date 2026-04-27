import Foundation
import Combine

@MainActor
final class ResultViewModel: ObservableObject {
    let parseResult: ScannerViewModel.ParseResult

    @Published var showShareSheet = false

    init(parseResult: ScannerViewModel.ParseResult) {
        self.parseResult = parseResult
    }

    var hasErrors: Bool {
        parseResult.validations.contains(where: { $0.level == .error })
    }

    var hasWarnings: Bool {
        parseResult.validations.contains(where: { $0.level == .warning })
    }

    var splitIncomplete: Bool {
        parseResult.messages.contains(where: { $0.contains("分割") })
    }

    var shareText: String { parseResult.formattedText }

    var shareFilename: String {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyyMMdd_HHmmss"
        return "prescription_\(formatter.string(from: Date())).txt"
    }
}
