import SwiftUI

struct ShareSheet: UIViewControllerRepresentable {
    let text: String
    let filename: String

    func makeUIViewController(context: Context) -> UIActivityViewController {
        let url = FileManager.default.temporaryDirectory.appendingPathComponent(filename)
        try? text.write(to: url, atomically: true, encoding: .utf8)
        return UIActivityViewController(activityItems: [url], applicationActivities: nil)
    }

    func updateUIViewController(_ uiViewController: UIActivityViewController, context: Context) {}
}
