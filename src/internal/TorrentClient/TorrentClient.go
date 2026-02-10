package torrentclient

// ========== Struct Defs =========== //

// stores state for a client
type TorrentClient struct {
	PeerID        [20]byte
	Port          uint16
	Uploaded      uint64
	Downloaded    uint64
	Left          uint64
	ActivePeers   []PeerInfo
	RateLimitUp   uint64
	RateLimitDown uint64
}

// stores peer info returned from trackers

type PeerInfo struct {
	PeerID         [20]byte `json:"peer_id"`
	IP             string   `json:"ip"`
	Port           uint16   `json:"port"`
	AmChoking      bool     `json:"am_choking"`
	AmInterested   bool     `json:"am_interested"`
	PeerChoking    bool     `json:"peer_choking"`
	PeerInterested bool     `json:"peer_interested"`
	LastSeen       int64    `json:"last_seen"` // Unix timestamp
}

type TrackerResponse struct {
	FailureReason string `json:"failure_reason"`
	Interval      int64  `json:"interval"`
	TrackerId     string `json:"tracker"`
	Complete      int64  `json:"complete"`
	Incomplete    int64  `json:"incomplete"`
	// Peers         []PeerInfo `json:"peers"`
	Peers string `json:"peers"`
}

// ========== Method Defs =========== //

func (t *TorrentClient) GetPeerStringId() string {
	return string(t.PeerID[:])
}

// func (t *TorrentClient) StartTorrent(torrentfile torrent.TorrentFile) {
// try all announce urls, use first working
// trackerUrls := torrentfile.BuildAllTrackerUrl(t.GetPeerStringId(), 1234)

// }

// func (t *TorrentClient) RequestTrackerServer(trackerUrl string)
