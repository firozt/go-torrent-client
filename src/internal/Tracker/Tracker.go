package tracker

import peers "github.com/firozt/go-torrent/src/internal/Peers"

type TrackerResponse struct {
	FailureReason string        `json:"failure_reason"`
	Interval      int64         `json:"interval"`
	TrackerId     string        `json:"tracker"`
	Complete      int64         `json:"complete"`
	Incomplete    int64         `json:"incomplete"`
	Peers         *[]peers.Peer // holds parsed info from peers blob
	RawPeers      string        `json:"peers"`
}
