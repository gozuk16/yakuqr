import XCTest
@testable import yakuqr_ios

final class JAHISValidatorTests: XCTestCase {

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

    func testValidate_ver4_valid_hasNoErrors() throws {
        let raw = RawQR(text: try readTestdata("ver4_single.txt"), errMsg: "")
        let (p, _) = JAHISParser.parse([raw])
        let errors = JAHISValidator.validate(p).filter { $0.level == .error }
        XCTAssertTrue(errors.isEmpty, "有効なVer.4データにエラーがあってはならない: \(errors.map(\.description))")
    }

    func testValidate_ver4_missingRecord1_hasError() {
        let p = JAHISPrescription(
            version: .v2_1,
            rawQRs: [RawQR(text: "dummy", errMsg: "")],
            records: [],
            recordMap: [:],
            splitInfos: []
        )
        let results = JAHISValidator.validate(p)
        XCTAssertTrue(results.contains(where: { $0.level == .error && $0.field.contains("レコード1") }))
    }

    func testValidate_ver4_emptyPatientName_hasError() {
        let record1 = JAHISRecord(type: "1", fields: ["1", "", ""])
        let p = JAHISPrescription(
            version: .v2_1,
            rawQRs: [RawQR(text: "dummy", errMsg: "")],
            records: [record1],
            recordMap: ["1": [record1]],
            splitInfos: []
        )
        let results = JAHISValidator.validate(p)
        XCTAssertTrue(results.contains(where: { $0.level == .error && $0.field.contains("患者氏名") }))
    }

    func testValidate_ver4_invalidDateFormat_hasError() {
        let record1 = JAHISRecord(type: "1", fields: ["1", "山田太郎", "1", "INVALID"])
        let p = JAHISPrescription(
            version: .v2_1,
            rawQRs: [RawQR(text: "dummy", errMsg: "")],
            records: [record1],
            recordMap: ["1": [record1]],
            splitInfos: []
        )
        let results = JAHISValidator.validate(p)
        XCTAssertTrue(results.contains(where: { $0.level == .error && $0.field.contains("生年月日") }))
    }

    func testValidate_ver4_missingDrugRecord_hasWarning() {
        let record1 = JAHISRecord(type: "1", fields: ["1", "山田太郎", "1", "19700101"])
        let p = JAHISPrescription(
            version: .v2_1,
            rawQRs: [RawQR(text: "dummy", errMsg: "")],
            records: [record1],
            recordMap: ["1": [record1]],
            splitInfos: []
        )
        let results = JAHISValidator.validate(p)
        XCTAssertTrue(results.contains(where: { $0.level == .warning && $0.field.contains("薬品情報") }))
    }

    func testValidate_ver2_valid_hasNoErrors() throws {
        let raw = RawQR(text: try readTestdata("ver2_single.txt"), errMsg: "")
        let (p, _) = JAHISParser.parse([raw])
        let errors = JAHISValidator.validate(p).filter { $0.level == .error }
        XCTAssertTrue(errors.isEmpty, "有効なVer.2データにエラーがあってはならない: \(errors.map(\.description))")
    }
}
