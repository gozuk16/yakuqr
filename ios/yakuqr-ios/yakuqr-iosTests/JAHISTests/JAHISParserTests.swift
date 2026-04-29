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

    func testParse_ver2_6_detectsVersion() throws {
        let raw = RawQR(text: try readTestdata("ver2_6_911split_1.txt"), errMsg: "")
        let (p, _) = JAHISParser.parse([raw])
        XCTAssertEqual(p.version, .v2_6)
    }

    func testParse_911split_3way_all_hasAllRps() throws {
        let raws = try [
            rawQR("ver2_6_911split3_1.txt"),
            rawQR("ver2_6_911split3_2.txt"),
            rawQR("ver2_6_911split3_3.txt"),
        ]
        let (p, _) = JAHISParser.parse(raws)
        XCTAssertEqual(p.version, .v2_6)
        XCTAssertEqual(p.recordMap["201"]?.count, 3, "Rp1〜Rp3の3件が必要")
        XCTAssertNil(p.recordMap["911"], "911行はRecordMapに残らない")
    }

    func testParse_911split_missingQR2_usesMaxCumulative() throws {
        // QR1+QR3のみ（QR2欠落）: 累積型のため QR3 がすべてのRpを含む
        let raws = try [rawQR("ver2_6_911split3_1.txt"), rawQR("ver2_6_911split3_3.txt")]
        let (p, _) = JAHISParser.parse(raws)
        XCTAssertEqual(p.recordMap["201"]?.count, 3, "累積型: QR2欠落でもRp1〜Rp3全て取得できる")
        XCTAssertNil(p.recordMap["911"])
    }

    func testParse_911split_qr1Only_has911InRecordMap() throws {
        // QR1のみ: 1パーツなので 911 パスに入らず、911行がRecordMapに残る
        let (p, _) = JAHISParser.parse([try rawQR("ver2_6_911split_1.txt")])
        XCTAssertNotNil(p.recordMap["911"], "QR1のみのとき911行がRecordMapに残る")
    }
}
