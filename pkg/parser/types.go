package parser

// Version はJAHIS規約のバージョンを表す。
type Version int

const (
	VersionUnknown Version = 0
	Version1_0     Version = 1
	Version1_1     Version = 2
	Version2_0     Version = 3
	Version2_1     Version = 4
	Version2_2     Version = 5
	Version2_3     Version = 6
	Version2_4     Version = 7
	// Version2_5 は欠番。JAHISTC08 は Ver.2.6 に対応するため Ver.2.5 は使用されない。
	Version2_6 Version = 8
)

func (v Version) String() string {
	switch v {
	case VersionUnknown:
		return "Unknown"
	case Version1_0:
		return "Ver.1.0"
	case Version1_1:
		return "Ver.1.1"
	case Version2_0:
		return "Ver.2.0"
	case Version2_1:
		return "Ver.2.1"
	case Version2_2:
		return "Ver.2.2"
	case Version2_3:
		return "Ver.2.3"
	case Version2_4:
		return "Ver.2.4"
	case Version2_6:
		return "Ver.2.6"
	default:
		return "Unknown"
	}
}

// RawQR は1枚の物理QRコードの生データを表す。
// ErrMsg が空なら Text にデコード済みテキストが入る。
// ErrMsg が非空なら読み取り失敗を示す（Text は空）。
type RawQR struct {
	Text   string
	ErrMsg string
}

// Record はJAHISの1レコードを表す。
type Record struct {
	Type   string   // レコード種別番号（先頭フィールド）
	Fields []string // 種別番号を含む全フィールド
}

// SplitInfo は分割QRの情報を保持する。
type SplitInfo struct {
	Current int // 現在のQR番号（1始まり）
	Total   int // 分割総数
}

// Prescription は解析済み処方箋データを表す。
type Prescription struct {
	Version    Version
	RawQRs     []RawQR             // 生QRデータ（成功・失敗含む）
	Records    []Record            // 全レコード（結合後）
	RecordMap  map[string][]Record // レコード種別 → レコードリスト
	SplitInfos []SplitInfo         // 分割QR情報（なければlen=0）
}
