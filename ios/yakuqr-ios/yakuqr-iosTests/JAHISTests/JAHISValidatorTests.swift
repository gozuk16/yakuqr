import XCTest
@testable import yakuqr_ios

final class JAHISValidatorTests: XCTestCase {

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

    func testValidate_ver4_valid_hasNoErrors() throws {
        let raw = try readTestdata("ver4_single.txt")
        let (p, _) = JAHISParser.parse([raw])
        let results = JAHISValidator.validate(p)
        let errors = results.filter { $0.level == .error }
        XCTAssertTrue(errors.isEmpty, "有効なVer.4データにエラーがあってはならない: \(errors.map(\.description))")
    }

    func testValidate_ver4_missingRecord1_hasError() {
        let prescription = JAHISPrescription(
            version: .v4,
            rawQRs: ["dummy"],
            records: [],
            recordMap: [:],
            splitInfos: []
        )
        let results = JAHISValidator.validate(prescription)
        XCTAssertTrue(results.contains(where: {
            $0.level == .error && $0.field.contains("レコード1")
        }), "レコード1欠落はERROR")
    }

    func testValidate_ver4_emptyPatientName_hasError() {
        let record1 = JAHISRecord(type: "1", fields: ["1", "", ""])
        let prescription = JAHISPrescription(
            version: .v4,
            rawQRs: ["dummy"],
            records: [record1],
            recordMap: ["1": [record1]],
            splitInfos: []
        )
        let results = JAHISValidator.validate(prescription)
        XCTAssertTrue(results.contains(where: {
            $0.level == .error && $0.field.contains("患者氏名")
        }), "患者氏名が空のときERROR")
    }

    func testValidate_ver4_invalidDateFormat_hasError() {
        let record1 = JAHISRecord(type: "1", fields: ["1", "山田太郎", "1", "INVALID"])
        let prescription = JAHISPrescription(
            version: .v4,
            rawQRs: ["dummy"],
            records: [record1],
            recordMap: ["1": [record1]],
            splitInfos: []
        )
        let results = JAHISValidator.validate(prescription)
        XCTAssertTrue(results.contains(where: {
            $0.level == .error && $0.field.contains("生年月日")
        }), "不正な日付フォーマットはERROR")
    }

    func testValidate_ver4_missingDrugRecord_hasWarning() {
        let record1 = JAHISRecord(type: "1", fields: ["1", "山田太郎", "1", "19700101"])
        let prescription = JAHISPrescription(
            version: .v4,
            rawQRs: ["dummy"],
            records: [record1],
            recordMap: ["1": [record1]],
            splitInfos: []
        )
        let results = JAHISValidator.validate(prescription)
        XCTAssertTrue(results.contains(where: {
            $0.level == .warning && $0.field.contains("薬品情報")
        }), "薬品情報なしはWARNING")
    }

    func testValidate_ver2_valid_hasNoErrors() throws {
        let raw = try readTestdata("ver2_single.txt")
        let (p, _) = JAHISParser.parse([raw])
        let results = JAHISValidator.validate(p)
        let errors = results.filter { $0.level == .error }
        XCTAssertTrue(errors.isEmpty, "有効なVer.2データにエラーがあってはならない: \(errors.map(\.description))")
    }
}
