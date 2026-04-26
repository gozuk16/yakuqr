import XCTest
@testable import yakuqr_ios

final class JAHISParserTests: XCTestCase {

    private func readTestdata(_ name: String) throws -> String {
        let thisFile = URL(fileURLWithPath: #file)
        let repoRoot = thisFile
            .deletingLastPathComponent() // JAHISTests/
            .deletingLastPathComponent() // yakuqr-iosTests/
            .deletingLastPathComponent() // yakuqr-ios/
            .deletingLastPathComponent() // ios/
            .deletingLastPathComponent() // yakuqr/
        let path = repoRoot.appendingPathComponent("testdata/\(name)").path
        return try String(contentsOfFile: path, encoding: .utf8)
    }

    func testParse_singleQR_ver4_detectsVersion() throws {
        let raw = try readTestdata("ver4_single.txt")
        let (p, _) = JAHISParser.parse([raw])
        XCTAssertEqual(p.version, .v4)
    }

    func testParse_singleQR_ver4_hasRecords() throws {
        let raw = try readTestdata("ver4_single.txt")
        let (p, _) = JAHISParser.parse([raw])
        XCTAssertFalse(p.records.isEmpty)
        XCTAssertNotNil(p.recordMap["1"], "レコード種別1（処方箋情報）が必要")
        XCTAssertNotNil(p.recordMap["201"], "レコード種別201（薬品情報）が必要")
    }

    func testParse_splitQR_combined() throws {
        let r1 = try readTestdata("ver4_split_1.txt")
        let r2 = try readTestdata("ver4_split_2.txt")
        let (p, _) = JAHISParser.parse([r1, r2])
        XCTAssertNotNil(p.recordMap["201"], "分割QR結合後にレコード201が必要")
        XCTAssertEqual(p.splitInfos.count, 2)
    }

    func testParse_splitQR_missing_hasWarning() throws {
        let r2 = try readTestdata("ver4_split_only2.txt")
        let (_, msgs) = JAHISParser.parse([r2])
        XCTAssertTrue(msgs.contains(where: { $0.contains("分割") }), "欠番の警告が必要")
    }

    func testParse_ver3_detectsVersion() throws {
        let raw = try readTestdata("ver3_single.txt")
        let (p, _) = JAHISParser.parse([raw])
        XCTAssertEqual(p.version, .v3)
    }

    func testParse_ver2_detectsVersion() throws {
        let raw = try readTestdata("ver2_single.txt")
        let (p, _) = JAHISParser.parse([raw])
        XCTAssertEqual(p.version, .v2)
    }

    func testParse_unknownVersion_fallsBackToVer4() {
        let raw = "99,unknown\n2,テスト,テスト,19900101,1,,,"
        let (p, msgs) = JAHISParser.parse([raw])
        XCTAssertEqual(p.version, .v4)
        XCTAssertTrue(msgs.contains(where: { $0.contains("Ver.4") }))
    }
}
