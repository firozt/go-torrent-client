// Package torrentclient that contains the client logic, high level package
package torrentclient

import (
	"fmt"
	"net/url"

	peers "github.com/firozt/go-torrent/src/internal/Peers"
	torrent "github.com/firozt/go-torrent/src/internal/Torrent"
	// torrent "github.com/firozt/go-torrent/src/internal/Torrent"
)

// ========== Struct Defs =========== //

// TorrentClient stores state for a client
type TorrentClient struct {
	PeerID        [20]byte
	Port          uint16
	Uploaded      uint64
	Downloaded    uint64
	Left          uint64
	ActivePeers   []peers.Peer
	RateLimitUp   uint64
	RateLimitDown uint64
}

// TrackerResponse stores peer info returned from trackers
type TrackerResponse struct {
	FailureReason string        `json:"failure_reason"`
	Interval      int64         `json:"interval"`
	TrackerID     string        `json:"tracker"`
	Complete      int64         `json:"complete"`
	Incomplete    int64         `json:"incomplete"`
	Peers         *[]peers.Peer // holds parsed info from peers blob
	RawPeers      string        `json:"peers"`
}

// ========== Method Defs =========== //

func (t *TorrentClient) GetPeerStringID() string {
	return string(t.PeerID[:])
}

func (t *TorrentClient) StartTorrent(torrentfile torrent.TorrentFile) {
	// try all announce urls, use first working
	trackerURLS := torrentfile.BuildAllTrackerURL(string(t.PeerID[:]), 12345)

	for _, tracker := range trackerURLS {
		_, trackerErr := t.getTrackerResponse(tracker)

		if trackerErr != nil {
			continue
		}
	}

}

// url can either point to a http server or a udp server
func (t *TorrentClient) getTrackerResponse(trackerURL string) (*TrackerResponse, error) {
	u, err := url.Parse(trackerURL)

	if err != nil {
		return nil, err
	}

	if u.Scheme == "udp" {

	}

	if u.Scheme == "http" {

	}

	return nil, fmt.Errorf("unknown url scheme - %s", u.Scheme)
}

func (t *TorrentClient) RequestTrackerServer(trackerUrl string)
