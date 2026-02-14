package torrentclient

import (
	peers "github.com/firozt/go-torrent/src/internal/Peers"
	// torrent "github.com/firozt/go-torrent/src/internal/Torrent"
)

// ========== Struct Defs =========== //

// stores state for a client
type TorrentClient struct {
	PeerID        [20]byte
	Port          uint16
	Uploaded      uint64
	Downloaded    uint64
	Left          uint64
	ActivePeers   []peers.PeerInfo
	RateLimitUp   uint64
	RateLimitDown uint64
}

// stores peer info returned from trackers

type TrackerResponse struct {
	FailureReason string            `json:"failure_reason"`
	Interval      int64             `json:"interval"`
	TrackerId     string            `json:"tracker"`
	Complete      int64             `json:"complete"`
	Incomplete    int64             `json:"incomplete"`
	Peers         *[]peers.PeerInfo // holds parsed info from peers blob
	rawPeers      string            `json:"peers"`
}

// ========== Method Defs =========== //

func (t *TorrentClient) GetPeerStringId() string {
	return string(t.PeerID[:])
}

// func [](t *TorrentClient) StartTorrent(torrentfile torrent.TorrentFile) {
// 	// try all announce urls, use first working
// 	trackerUrls := torrentfile.BuildAllTrackerUrl(t.GetPeerStringId(), 1234)
//
// }

// func (t *TorrentClient) getTrackerResponse() (, error) {

// }

// func (t *TorrentClient) RequestTrackerServer(trackerUrl string)
