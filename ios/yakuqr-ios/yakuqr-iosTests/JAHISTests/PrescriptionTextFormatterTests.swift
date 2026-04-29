import XCTest
@testable import yakuqr_ios

final class PrescriptionTextFormatterTests: XCTestCase {

    private func readTestdata(_ name: String) throws -> String {
        let thisFile = URL(fileURLWithPath: #file)
        let repoRoot = thisFile
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .deletingLastPathComponent()
        return try String(
            contentsOfFile: repoRoot.appendingPathComponent("testdata/\(name)").path,
            encoding: .utf8
        )
    }

    func testFormat_ver4_containsRawQR() throws {
        let raw = RawQR(text: "JAHISTC04,1\n1,山田太郎,1,19700101,,,,,,\n201,1,アムロジピン錠,1,錠,4,xxx,1", errMsg: "")
        let (p, _) = JAHISParser.parse([raw])
        let text = PrescriptionTextFormatter.format(p, validations: JAHISValidator.validate(p))
        XCTAssertTrue(text.contains("RAW QR DATA"))
        XCTAssertTrue(text.contains("JAHISTC04"))
    }

    func testFormat_ver4_containsValidation() throws {
        let raw = RawQR(text: "JAHISTC04,1\n1,,,", errMsg: "")
        let (p, _) = JAHISParser.parse([raw])
        let text = PrescriptionTextFormatter.format(p, validations: JAHISValidator.validate(p))
        XCTAssertTrue(text.contains("VALIDATION"))
        XCTAssertTrue(text.contains("ERROR"))
    }

    func testFormat_ver4_noValidationErrors_containsOkMessage() {
        let record1 = JAHISRecord(type: "1", fields: ["1", "山田太郎", "1", "19700101"])
        let record201 = JAHISRecord(type: "201", fields: ["201", "1", "テスト薬", "1", "錠", "4", "xxx", "1"])
        let p = JAHISPrescription(
            version: .v2_1,
            rawQRs: [RawQR(text: "dummy", errMsg: "")],
            records: [record1, record201],
            recordMap: ["1": [record1], "201": [record201]],
            splitInfos: []
        )
        let text = PrescriptionTextFormatter.format(p, validations: [])
        XCTAssertTrue(text.contains("問題は検出されませんでした"))
    }

    func testFormat_ver2_containsPatientName() throws {
        let raw = RawQR(text: try readTestdata("ver2_single.txt"), errMsg: "")
        let (p, _) = JAHISParser.parse([raw])
        let text = PrescriptionTextFormatter.format(p, validations: [])
        XCTAssertTrue(text.contains("RAW QR DATA"))
        XCTAssertTrue(text.contains("Ver.1.1"))
        XCTAssertTrue(text.contains("テスト次郎"))
    }
}
