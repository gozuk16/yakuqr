enum JAHISVersion: Int {
    case unknown = 0
    case v1_0 = 1
    case v1_1 = 2
    case v2_0 = 3
    case v2_1 = 4
    case v2_6 = 8

    var displayName: String {
        switch self {
        case .v1_0: return "Ver.1.0"
        case .v1_1: return "Ver.1.1"
        case .v2_0: return "Ver.2.0"
        case .v2_1: return "Ver.2.1"
        case .v2_6: return "Ver.2.6"
        case .unknown: return "Unknown"
        }
    }
}

struct RawQR {
    let text: String
    let errMsg: String
    var isSuccess: Bool { errMsg.isEmpty }
}

struct JAHISRecord {
    let type: String
    let fields: [String]
}

struct JAHISSplitInfo {
    let current: Int
    let total: Int
}

struct JAHISPrescription {
    let version: JAHISVersion
    let rawQRs: [RawQR]
    let records: [JAHISRecord]
    let recordMap: [String: [JAHISRecord]]
    let splitInfos: [JAHISSplitInfo]
}

enum ValidationLevel: Int, Comparable {
    case error = 0
    case warning = 1
    case info = 2

    static func < (lhs: ValidationLevel, rhs: ValidationLevel) -> Bool {
        lhs.rawValue < rhs.rawValue
    }

    var displayName: String {
        switch self {
        case .error: return "ERROR"
        case .warning: return "WARNING"
        case .info: return "INFO"
        }
    }
}

struct JAHISValidationResult {
    let level: ValidationLevel
    let field: String
    let message: String

    var description: String { "[\(level.displayName)] \(field): \(message)" }
}
