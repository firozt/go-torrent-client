package peers

import (
	"encoding/binary"
	"fmt"
	"net"
)

type Peer struct {
	// peerID         [20]byte
	ipv4Addr net.IP
	port     uint16
	// amChoking      bool
	// amInterested   bool
	// peerChoking    bool
	// peerInterested bool
	// lastSeen       int64
}

var ErrInvalidPeerBlob = fmt.Errorf("invalid Peer Blob")

/*
Takes a binary blop representing a list of peers
A peer object is 6 bytes long
Structure of peer is:

bytes	0                   4	      6

	|----|----|----|----|----|----|
	---peer ipv4 addr--- peer port

ip addr is stored in network order (big endian)
*/
func MakePeer(peerBlob []byte) ([]Peer, error) {
	res := make([]Peer, len(peerBlob)/6)

	// blop must be a multiple of 6
	if len(peerBlob)%6 != 0 {
		return nil, ErrInvalidPeerBlob
	}

	for i := 0; i < len(res)/6; i++ {
		startIdx := 6 * i
		// account for network byte order
		var port uint16
		binary.BigEndian.PutUint16(peerBlob[startIdx:startIdx+4], port)

		res = append(res, Peer{
			ipv4Addr: net.IP(peerBlob[startIdx+4:]),
			port:     port,
		})
	}
	return res, nil
}
