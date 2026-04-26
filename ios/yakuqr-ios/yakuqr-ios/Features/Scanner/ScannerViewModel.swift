import Foundation
import Combine
import Vision
import UIKit

@MainActor
final class ScannerViewModel: ObservableObject {
    @Published var scannedQRs: [String] = []
    @Published var parseResult: ParseResult? = nil
    @Published var showPhotoPermissionAlert = false

    private var seenQRs: Set<String> = []
    private var cancellables: Set<AnyCancellable> = []

    struct ParseResult {
        let prescription: JAHISPrescription
        let validations: [JAHISValidationResult]
        let messages: [String]
        let formattedText: String
    }

    func bind(cameraService: CameraService) {
        cameraService.detectedQRPublisher
            .receive(on: DispatchQueue.main)
            .sink { [weak self] value in self?.addQR(value) }
            .store(in: &cancellables)
    }

    func addQR(_ value: String) {
        guard !seenQRs.contains(value) else { return }
        seenQRs.insert(value)
        scannedQRs.append(value)
    }

    func parse() {
        let (prescription, msgs) = JAHISParser.parse(scannedQRs)
        let validations = JAHISValidator.validate(prescription)
        let text = PrescriptionTextFormatter.format(prescription, validations: validations)
        parseResult = ParseResult(
            prescription: prescription,
            validations: validations,
            messages: msgs,
            formattedText: text
        )
    }

    func reset() {
        scannedQRs = []
        seenQRs = []
        parseResult = nil
    }

    func processImage(_ image: UIImage) {
        guard let cgImage = image.cgImage else { return }
        let request = VNDetectBarcodesRequest { [weak self] req, _ in
            guard let self else { return }
            let values = (req.results as? [VNBarcodeObservation] ?? [])
                .filter { $0.symbology == .qr }
                .compactMap(\.payloadStringValue)
            DispatchQueue.main.async {
                values.forEach { self.addQR($0) }
            }
        }
        let handler = VNImageRequestHandler(cgImage: cgImage)
        try? handler.perform([request])
    }
}
