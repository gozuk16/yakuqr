import XCTest
@testable import yakuqr_ios

final class JAHISParserTests: XCTestCase {

    private func readTestdata(_ name: String) throws -> String {
        let thisFile = URL(fileURLWithPath: #file)
        let repoRoot = thisFile
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .deletingLastPathComponent()
        let path = repoRoot.appendingPathComponent("testdata/\(name)").path
        return try String(contentsOfFile: path, encoding: .utf8)
    }

    private func rawQR(_ name: String) throws -> RawQR {
        RawQR(text: try readTestdata(name), errMsg: "")
    }

    func testParse_singleQR_ver4_detectsVersion() throws {
        let (p, _) = JAHISParser.parse([try rawQR("ver4_single.txt")])
        XCTAssertEqual(p.version, .v2_1)
    }

    func testParse_singleQR_ver4_hasRecords() throws {
        let (p, _) = JAHISParser.parse([try rawQR("ver4_single.txt")])
        XCTAssertFalse(p.records.isEmpty)
        XCTAssertNotNil(p.recordMap["1"])
        XCTAssertNotNil(p.recordMap["201"])
    }

    func testParse_splitQR_combined() throws {
        let (p, _) = JAHISParser.parse([try rawQR("ver4_split_1.txt"), try rawQR("ver4_split_2.txt")])
        XCTAssertNotNil(p.recordMap["201"])
        XCTAssertEqual(p.splitInfos.count, 2)
    }

    func testParse_splitQR_missing_hasWarning() throws {
        let (_, msgs) = JAHISParser.parse([try rawQR("ver4_split_only2.txt")])
        XCTAssertTrue(msgs.contains(where: { $0.contains("分割") }))
    }

    func testParse_ver3_detectsVersion() throws {
        let (p, _) = JAHISParser.parse([try rawQR("ver3_single.txt")])
        XCTAssertEqual(p.version, .v2_0)
    }

    func testParse_ver2_detectsVersion() throws {
        let (p, _) = JAHISParser.parse([try rawQR("ver2_single.txt")])
        XCTAssertEqual(p.version, .v1_1)
    }

    func testParse_unknownVersion_fallsBackToVer2_1() {
        let raw = RawQR(text: "99,unknown\n2,テスト,テスト,19900101,1,,,", errMsg: "")
        let (p, msgs) = JAHISParser.parse([raw])
        XCTAssertEqual(p.version, .v2_1)
        XCTAssertTrue(msgs.contains(where: { $0.contains("Ver.2.1") }))
    }
}
