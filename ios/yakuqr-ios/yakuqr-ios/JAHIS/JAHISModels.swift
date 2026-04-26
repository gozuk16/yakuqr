enum JAHISVersion: Int {
    case unknown = 0
    case v2 = 2
    case v3 = 3
    case v4 = 4

    var displayName: String {
        switch self {
        case .v2: return "Ver.2"
        case .v3: return "Ver.3"
        case .v4: return "Ver.4"
        case .unknown: return "Unknown"
        }
    }
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
    let rawQRs: [String]
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
