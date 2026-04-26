import XCTest
@testable import yakuqr_ios

@MainActor
final class ScannerViewModelTests: XCTestCase {

    func testAddQR_deduplicates() {
        let vm = ScannerViewModel()
        vm.addQR("QR_DATA_A")
        vm.addQR("QR_DATA_A")
        XCTAssertEqual(vm.scannedQRs.count, 1, "重複QRは追加されない")
    }

    func testAddQR_multipleDistinct() {
        let vm = ScannerViewModel()
        vm.addQR("QR_DATA_A")
        vm.addQR("QR_DATA_B")
        XCTAssertEqual(vm.scannedQRs.count, 2)
    }

    func testReset_clearsState() {
        let vm = ScannerViewModel()
        vm.addQR("QR_DATA_A")
        vm.reset()
        XCTAssertTrue(vm.scannedQRs.isEmpty)
        XCTAssertNil(vm.parseResult)
    }

    func testParse_withVer4Data_setsParseResult() throws {
        let thisFile = URL(fileURLWithPath: #file)
        let repoRoot = thisFile
            .deletingLastPathComponent() // JAHISTests/
            .deletingLastPathComponent() // yakuqr-iosTests/
            .deletingLastPathComponent() // yakuqr-ios/
            .deletingLastPathComponent() // ios/
            .deletingLastPathComponent() // repo root
        let raw = try String(
            contentsOfFile: repoRoot.appendingPathComponent("testdata/ver4_single.txt").path,
            encoding: .utf8
        )
        let vm = ScannerViewModel()
        vm.addQR(raw)
        vm.parse()
        XCTAssertNotNil(vm.parseResult)
        XCTAssertEqual(vm.parseResult?.prescription.version, .v4)
    }

    func testParse_emptyQRs_setsParseResult() {
        let vm = ScannerViewModel()
        vm.parse()
        XCTAssertNotNil(vm.parseResult)
    }
}
