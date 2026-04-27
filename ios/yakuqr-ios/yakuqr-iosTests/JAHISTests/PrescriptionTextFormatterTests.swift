import XCTest
@testable import yakuqr_ios

final class PrescriptionTextFormatterTests: XCTestCase {

    private func readTestdata(_ name: String) throws -> String {
        let thisFile = URL(fileURLWithPath: #file)
        let repoRoot = thisFile
            .deletingLastPathComponent() // JAHISTests/
            .deletingLastPathComponent() // yakuqr-iosTests/
            .deletingLastPathComponent() // yakuqr-ios/
            .deletingLastPathComponent() // ios/
            .deletingLastPathComponent() // repo root
        return try String(
            contentsOfFile: repoRoot.appendingPathComponent("testdata/\(name)").path,
            encoding: .utf8
        )
    }

    func testFormat_ver4_containsRawQR() throws {
        let raw = "JAHISTC04,1\n1,山田太郎,1,19700101,,,,,,\n201,1,アムロジピン錠,1,錠,4,xxx,1"
        let (p, _) = JAHISParser.parse([raw])
        let validations = JAHISValidator.validate(p)
        let text = PrescriptionTextFormatter.format(p, validations: validations)
        XCTAssertTrue(text.contains("RAW QR DATA"), "生QRセクションが必要")
        XCTAssertTrue(text.contains("JAHISTC04"), "生QRデータが含まれる必要がある")
    }

    func testFormat_ver4_containsValidation() throws {
        let raw = "JAHISTC04,1\n1,,,"
        let (p, _) = JAHISParser.parse([raw])
        let validations = JAHISValidator.validate(p)
        let text = PrescriptionTextFormatter.format(p, validations: validations)
        XCTAssertTrue(text.contains("VALIDATION"), "バリデーションセクションが必要")
        XCTAssertTrue(text.contains("ERROR"), "エラーが含まれる必要がある")
    }

    func testFormat_ver4_noValidationErrors_containsOkMessage() {
        let record1 = JAHISRecord(type: "1", fields: ["1", "山田太郎", "1", "19700101"])
        let record201 = JAHISRecord(type: "201", fields: ["201", "1", "テスト薬", "1", "錠", "4", "xxx", "1"])
        let p = JAHISPrescription(
            version: .v4,
            rawQRs: ["dummy"],
            records: [record1, record201],
            recordMap: ["1": [record1], "201": [record201]],
            splitInfos: []
        )
        let text = PrescriptionTextFormatter.format(p, validations: [])
        XCTAssertTrue(text.contains("問題は検出されませんでした"))
    }

    func testFormat_ver2_containsPatientName() throws {
        let raw = try readTestdata("ver2_single.txt")
        let (p, _) = JAHISParser.parse([raw])
        let text = PrescriptionTextFormatter.format(p, validations: [])
        XCTAssertTrue(text.contains("RAW QR DATA"), "生QRセクションが必要")
        XCTAssertTrue(text.contains("Ver.2"), "バージョン表示が必要")
        XCTAssertTrue(text.contains("山田次郎"), "患者氏名が含まれる必要がある")
    }
}
