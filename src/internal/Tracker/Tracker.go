// Package tracker helps to deal with response from tracker server
package tracker

import (
	"fmt"

	peers "github.com/firozt/go-torrent/src/internal/Peers"
)

type TrackerResponse struct {
	FailureReason string        `json:"failure_reason"`
	Interval      int64         `json:"interval"`
	TrackerID     string        `json:"tracker"`
	Complete      int64         `json:"complete"`
	Incomplete    int64         `json:"incomplete"`
	peers         *[]peers.Peer // holds parsed info from peers blob
	RawPeers      string        `json:"peers"`
}

// GetPeers gets peers and if non existant will generate from raw peers
// May return a raw peers does not exist error
func (t *TrackerResponse) GetPeers() (*[]peers.Peer, error) {
	if len(t.RawPeers) == 0 {
		return nil, fmt.Errorf("peers does not exist for this variable")
	}
	val, err := peers.MakePeer([]byte(t.RawPeers))
	if err != nil {
		return nil, err
	}
	return &val, nil
}
