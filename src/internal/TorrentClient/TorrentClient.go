// Package torrentclient that contains the client logic, high level package
package torrentclient

import (
	"fmt"
	"net/http"
	"net/url"

	bencodeparser "github.com/firozt/go-torrent/src/internal/BencodeParser"
	peers "github.com/firozt/go-torrent/src/internal/Peers"
	torrent "github.com/firozt/go-torrent/src/internal/Torrent"
	tracker "github.com/firozt/go-torrent/src/internal/Tracker"
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
func (t *TorrentClient) getTrackerResponse(trackerURL string) (*tracker.TrackerResponse, error) {
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

func (t TorrentClient) handleHTTPScheme(httpURL *url.URL) (*tracker.TrackerResponse, error) {
	if httpURL.Scheme != "http" {
		return nil, fmt.Errorf("url provided is not a http url instead is %s", httpURL.Scheme)
	}

	// we need to make a request, and cancel out if it hangs as server may be down
	resp, err := http.Get(httpURL.String())

	if err != nil {
		return nil, err
	}

	// read and parse body
	defer resp.Body.Close()
	trackerResponse := &tracker.TrackerResponse{}

	err = bencodeparser.Read(resp.Body, trackerResponse)

	if err != nil {
		return nil, err
	}

	return trackerResponse, nil
}

// func (t *TorrentClient) RequestTrackerServer(trackerUrl string) {}
