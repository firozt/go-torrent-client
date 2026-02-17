// Package torrentclient that contains the client logic, high level package
package torrentclient

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/rand/v2"
	"net/http"
	"net/url"
	"time"

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
		return t.handleHTTPScheme(u)
	}

	return nil, fmt.Errorf("unknown url scheme - %s", u.Scheme)
}

func (t TorrentClient) handleHTTPScheme(httpURL *url.URL) (*tracker.TrackerResponse, error) {
	if httpURL.Scheme != "http" {
		return nil, fmt.Errorf("url provided is not a http url instead is %s", httpURL.Scheme)
	}

	// we need to make a request, and cancel out if it hangs as server may be down
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(httpURL.String())

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

/*
=============================================================================================

	Summary:

	HandleUDPScheme sends an announce packet to a tracker server via UDP
	all values are sent in network byte order (big endian)
	The connect request looks like the followint (16 bytes)

bytes	0	 2	  4 	   6	    8	    10	      12       14	16

	|--------|--------|--------|--------|--------|--------|--------|--------|

hexval   0x0000    0x0417   0x2710  0x1980 | 0x0000   0x0000  |    uint32       |

label	|---------connection_id------------|------Action------|-Transaction_id--|

	Meaning:

	connection_id ->  fixed constant defined by the protocol 64bit int
	action -> 	  action the sendee wants to accomplish (0 for connect), 32bit uint
	transaction_id -> randomly generated uint32 value to match response request

=============================================================================================
*/
func (t TorrentClient) handleUDPScheme(udpURL *url.URL) (*tracker.TrackerResponse, error) {
	if udpURL.Scheme != "udp" {
		return nil, fmt.Errorf("invalid scheme, wanted udp got %s", udpURL.Scheme)
	}

	transaction_id := uint32(rand.IntN(math.MaxUint32)) // 4 bytes
	action := uint32(0x1)                               // 4 byte
	connection_id := []bytes(0x41727101980)             // 8byte

	msg := make([]byte, 16)

	msg[0] = byte(binary.BigEndian.Uint64(connection_id))
	return nil, nil

}
