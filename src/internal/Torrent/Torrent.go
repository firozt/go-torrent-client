package torrent

// ============ Struct Defs  ============ //

type TorrentFile struct {
	Name         string
	Announce     string
	AnnounceList []string
	InfoHash     [20]byte
	CreationDate uint64
	PieceLength  uint64
	Pieces       [][20]byte
}

type TorrentFileSFM struct {
	TorrentFile
	Length uint64
}

type TorrentFileMFM struct {
	TorrentFile
	Files []RawTorrentFileField
}

type Torrent interface {
	IsMultiFile() bool
}

// ============ Raw Data Structs  ============ //

type RawTorrentInfo struct {
	Name        string                `bencode:"name" json:"name"`
	Length      int64                 `bencode:"length" json:"length"`
	PieceLength int64                 `bencode:"piece length" json:"piece length"`
	Piece       string                `bencode:"pieces" json:"pieces"`
	Files       []RawTorrentFileField `bencode:"files" json:"files"`
}

type RawTorrentFileField struct {
	Path   []string `bencode:"path" json:"path"`
	Length int64    `bencode:"length" json:"length"`
}

type RawTorrentData struct {
	InfoHash     [20]byte       `bencode:"info hash" json:"info_hash"`
	Announce     string         `bencode:"announce" json:"announce"`
	AnnounceList [][]any        `bencode:"announce list" json:"announce-list"`
	CreationDate int64          `bencode:"creation date" json:"creation date"`
	Info         RawTorrentInfo `bencode:"info" json:"info"`
}

// ============ Methods  ============ //

func (TorrentFile) IsMultiFile() bool {
	panic("This struct should not be used to hold actual data, but should be abstract")
}
func (TorrentFileMFM) IsMultiFile() bool { return true }
func (TorrentFileSFM) IsMultiFile() bool { return false }
