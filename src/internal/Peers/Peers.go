// Package peers contains peer struct and the only way of instantiating it via Make function, that validates
package peers

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
)

type Peer struct {
	ipv4Addr net.IP
	port     uint16
	PeerID   [20]byte
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

	for i := range len(res) {
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

func (p Peer) IP() net.IP {
	return p.ipv4Addr
}

func (p Peer) Port() uint16 {
	return p.port
}

func (p Peer) Address() string {
	return p.ipv4Addr.String() + ":" + strconv.Itoa(int(p.port))
}

// PeerHandshake represents the initial messages given in the peer protocol
type PeerHandshake struct {
	StrLen       uint8
	ProtocolName string
	Reserved     [8]byte
	InfoHash     [20]byte
	PeerID       [20]byte
}

func NewBitTorrentProtocolHandshake(infoHash, peerID [20]byte) *PeerHandshake {
	return &PeerHandshake{
		StrLen:       18,
		ProtocolName: "BitTorrent protocol",
		Reserved:     [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		PeerID:       peerID,
	}
}

// SerializePeerHandshake builds the message defined in the bittorrent spec for initialising a peer connection
// message structure -> <strlen uint8><pstr 19byte><reserved 8 bytes><info_hash 20 bytes><peer_id 20 bytes>
func (p PeerHandshake) SerializePeerHandshake() []byte {
	PEER_HANDSHAKE_MSG_LENGTH := 68
	buf := make([]byte, PEER_HANDSHAKE_MSG_LENGTH)

	buf[0] = p.StrLen // 19 in hex
	copy(buf[1:], []byte(p.ProtocolName))
	copy(buf[20:], p.Reserved[:])
	copy(buf[28:], p.InfoHash[:])
	copy(buf[48:], p.PeerID[:])

	return buf
}

// DeserializePeerHandshake takes the raw message from a client and parses it into a PeerHandshake struct
// if the struct does not follow the bittorrent protocol for this message an error is returned
func DeserializePeerHandshake(raw [68]byte) (*PeerHandshake, error) {
	// make message

	var infoHash [20]byte
	var peerID [20]byte
	var Reserved [8]byte

	copy(Reserved[:], raw[20:28])
	copy(infoHash[:], raw[28:48])
	copy(peerID[:], raw[48:])

	msg := PeerHandshake{
		StrLen:       uint8(raw[0]),
		ProtocolName: string(raw[1:20]),
		Reserved:     Reserved,
		InfoHash:     infoHash,
		PeerID:       peerID,
	}

	// validate fields

	if msg.StrLen != 19 {
		return nil, fmt.Errorf("field StrLen is not 19 instead %d", msg.StrLen)
	}

	if msg.ProtocolName != "BitTorrent protocol" {
		return nil, fmt.Errorf("protocol name is not 'BitTorrent protocol' instead is %s", msg.ProtocolName)
	}

	return &msg, nil
}
