// Package peers contains peer struct and the only way of instantiating it via Make function, that validates
package peers

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
)

// Peer represents a Peer recieved from a tracking serverm
// Note that PeerID may be empty given that they have not yet
// displayed this information, typically when tracking server returns a response
// more information can be found on https://www.bittorrent.org/beps/bep_0003.html
type Peer struct {
	ipv4Addr net.IP
	port     uint16
	PeerID   [20]byte
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

// PeerHandshake represents the handshake message exchanged between peers
// when a connection is first established. It must be the first message
// sent by both sides of the connection. The handshake is 68 bytes total.
// See: https://www.bittorrent.org/beps/bep_0003.html
type PeerHandshake struct {
	// The size of the string that depicts the protocol name
	// For BitTorrent this is alway 19
	StrLen uint8
	// The protocol identifier string
	// For the BitTorrent Protocol this is always 'BitTorrent Protocol'
	ProtocolName string
	// Reserved bytes, should be all 0x00's
	Reserved [8]byte
	// InfoHash of the torrentfile that both wanting to upload/download
	InfoHash [20]byte
	// Client Peer identifier
	PeerID [20]byte
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

	copy(Reserved[:], raw[20:28]) // reserved bytes
	copy(infoHash[:], raw[28:48]) // info hash
	copy(peerID[:], raw[48:])     // peerid

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

// Message represents a message that is sent/recieved during p2p communications
type Message struct {
	ID      uint8
	Payload []byte
}

// PeerConn represents and manages the actual connection from a peer to another peer
type PeerConn struct {
	peer        Peer
	conn        net.Conn
	sendChannel chan Message
}

func MakePeerConn(peer Peer, conn net.Conn) *PeerConn {
	return &PeerConn{
		peer:        peer,
		conn:        conn,
		sendChannel: make(chan Message),
	}
}

// PeerHandshakeProtocol attempts to start a connection to a peer using the peer communications protocol
// this is always done via tcp or utp
func PeerHandshakeProtocol(peer Peer, infoHash [20]byte, peerId [20]byte) (*net.TCPConn, error) {
	if len(peer.IP()) == 0 || peer.Port() == 0 {
		return nil, fmt.Errorf("peer is malformed - %s", peer.Address())
	}

	// build initHandshakeMsg
	initHandshakeMsg := NewBitTorrentProtocolHandshake(infoHash, peerId)

	// attempt to connect, 5 second timeout
	conn, err := net.DialTimeout("tcp", peer.Address(), 5*time.Second)
	if err != nil {
		return nil, err
	}

	// send init msg
	conn.Write(initHandshakeMsg.SerializePeerHandshake())

	// wait for a response for 5 seconds then timeout

	readBuf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(readBuf)
	if err != nil {
		return nil, err
	}

	if n != 68 {
		conn.Close()
		return nil, fmt.Errorf("number of bytes returned is not 68, the length of the expected response instead its %d", n)
	}

	peerHandshakeResponse, err := DeserializePeerHandshake([68]byte(readBuf))
	if err != nil {
		conn.Close()
		return nil, err
	}

	if peerHandshakeResponse.InfoHash != infoHash {
		conn.Close()
		return nil, fmt.Errorf("the infohash returned in the handshake are not equivilant, expected %x, got %x", infoHash, peerHandshakeResponse.InfoHash)
	}

	// convert from net.conn interface to tcpconn obj
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return nil, fmt.Errorf("expected *net.TCPConn, got type %T", conn)
	}

	return tcpConn, nil
}

func ContactPeers(peers []Peer, infoHash [20]byte, peerId [20]byte) ([]PeerConn, error) {
	var errs string 
	var res []PeerConn
	for _, peer := range peers {
		// call peer
		conn, err := PeerHandshakeProtocol(peer, infoHash, peerId)		
		if err != nil {
			fmt.Printf("[LOG] failed to contact peer %+v\n", peer)
			errs += fmt.Sprintf("failed to connect to peer %s on port %d\n", peer.ipv4Addr.String(), peer.port) 
			continue
		}

		// success
		fmt.Printf("[LOG] successfully contacted peer %+v\n", peer)
		peerconns := MakePeerConn(peer, conn)
		res = append(res, *peerconns)
	}

	if len(errs) >= 0 {
		return nil, fmt.Errorf("could not contact some peers: \n%s", errs)
	}

	return  nil, nil
}

func (pc PeerConn) StartLoop() {
	defer pc.conn.Close() // close conn after loop ends

	wg := sync.WaitGroup{}

	wg.Go(pc.readLoop)
	wg.Go(pc.writeLoop)

	wg.Wait()
}

func (pc PeerConn) readLoop() {

}

func (pc PeerConn) writeLoop() {

}
