// Package peers contains peer struct and the only way of instantiating it via Make function, that validates
package peers

import (
	"encoding/binary"
	"fmt"
	"net"
)

type Peer struct {
	ipv4Addr net.IP
	port     uint16
	// amChoking      bool
	// amInterested   bool
	// peerChoking    bool
	// peerInterested bool
	// lastSeen       int64
}

// ErrInvalidPeerBlob occurs when raw peer blob data is malformed
var ErrInvalidPeerBlob = fmt.Errorf("invalid Peer Blob")

// MakePeer parses a binary blob representing a list of peers.
// Each peer is represented by 6 bytes in the following format:
//
// Bytes 0–3: IPv4 address (network order / big-endian)
// Bytes 4–5: TCP port (network order / big-endian)
//
// Example memory layout for one peer:
//
// | byte0 | byte1 | byte2 | byte3 | byte4 | byte5 |
// |---------------IP--------------|-----Port------|
//
// MakePeers returns a slice of Peer structs parsed from the blob.
func MakePeer(peerBlob []byte) ([]Peer, error) {
	peerBlobSize := 6
	portOffset := 4

	// blob must be a multiple of 6
	if len(peerBlob)%peerBlobSize != 0 {
		return nil, ErrInvalidPeerBlob
	}

	res := make([]Peer, len(peerBlob)/peerBlobSize)
	insertPos := 0

	for i := 0; i < len(res); i++ {
		startIdx := peerBlobSize * i
		// account for network byte order
		res[insertPos] = Peer{
			ipv4Addr: net.IP(peerBlob[startIdx : startIdx+portOffset]),
			port:     binary.BigEndian.Uint16(peerBlob[startIdx+portOffset : startIdx+peerBlobSize]),
		}

		insertPos++
	}
	return res, nil
}
