package torrent

// ============ Struct Defs  ============ //

// flattened torrentfile struct with better typing, enforcing field values types
type TorrentFile struct {
	Name         string
	Announce     []string
	InfoHash     [20]byte
	CreationDate uint64
	PieceLength  uint64
	Pieces       [][20]byte
	Length       uint64
	Files        []TorrentFileField
}

// ============ Raw Data Structs  ============ //

// raw direct representation of the bencode struct
type RawTorrentInfo struct {
	Name        string             `bencode:"name" json:"name"`
	Length      int64              `bencode:"length" json:"length"`
	PieceLength int64              `bencode:"piece length" json:"piece length"`
	Piece       string             `bencode:"pieces" json:"pieces"`
	Files       []TorrentFileField `bencode:"files" json:"files"`
}

type TorrentFileField struct {
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

func (t *TorrentFile) IsMultiFile() bool {
	if len(t.Files) > 0 && len(t.Files[0].Path) != 0 {
		return true
	}

	if t.Length != 0 {
		return false
	}

	panic("Torrentfile is neither SFM or MFM")
}

// func (t TorrentFile) BuildTrackerURL() (string, error) {
// urlParsed, err := url.Parse(t.Announce)
// }
