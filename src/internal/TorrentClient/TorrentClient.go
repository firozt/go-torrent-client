// Package torrentclient that contains the client logic, high level package
package torrentclient

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	bencodeparser "github.com/firozt/go-torrent/src/internal/BencodeParser"
	peers "github.com/firozt/go-torrent/src/internal/Peers"
	torrent "github.com/firozt/go-torrent/src/internal/Torrent"
	tracker "github.com/firozt/go-torrent/src/internal/Tracker"
)

// ========== Struct Defs =========== //

// TorrentClient stores state for a client, one to one mapping with torrentfile
type TorrentClient struct {
	peerID      [20]byte
	port        uint16
	uploaded    uint64
	downloaded  uint64
	left        uint64
	activePeers []peers.Peer
	key         uint32
	// RateLimitUp   uint64
	// RateLimitDown uint64
}

// ========== Method Defs =========== //

func NewTorrentClient(port uint16) *TorrentClient {
	return &TorrentClient{
		peerID:      random20Bytes(),
		port:        port,
		uploaded:    0,
		downloaded:  0,
		left:        0,
		activePeers: []peers.Peer{},
		key:         randomUint32(),
		// RateLimitUp:
		// RateLimitDown:
	}
}

func random20Bytes() [20]byte {
	var b [20]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic("random 20 bytes panic")
	}
	return b
}

func (t *TorrentClient) GetPeerStringID() string {
	return string(t.peerID[:])
}

func (t *TorrentClient) StartTorrent(torrentfile torrent.TorrentFile) {

	var trackerResponse *tracker.TrackerResponse
	var err error

	for _, tracker := range torrentfile.Announce {
		trackerResponse, err = t.getTrackerResponse(tracker, &torrentfile)

		if err == nil {
			break
		}
	}

	if trackerResponse == nil {
		panic("No valid tracker annoucne responses")
	}

}

// url can either point to a http server or a udp server
func (t *TorrentClient) getTrackerResponse(trackerURL string, torrentFile *torrent.TorrentFile) (*tracker.TrackerResponse, error) {
	u, err := url.Parse(trackerURL)

	if err != nil {
		return nil, err
	}

	if u.Scheme == "udp" {
		return t.udpHandshakeProtocol(u, torrentFile)
	}

	if u.Scheme == "http" || u.Scheme == "https" {
		return t.httpHandshakeProtocol(u, torrentFile)
	}

	return nil, fmt.Errorf("unknown url scheme - %s", u.Scheme)
}

func (t TorrentClient) httpHandshakeProtocol(httpURL *url.URL, torrentFile *torrent.TorrentFile) (*tracker.TrackerResponse, error) {
	if httpURL.Scheme != "http" && httpURL.Scheme != "https" {
		return nil, fmt.Errorf("url provided is not a http url instead is %s", httpURL.Scheme)
	}

	// add params to url
	fullTrackerURL, _ := torrentFile.BuildTrackerURL(httpURL.String(), string(t.peerID[:]), 12345)

	// we need to make a request, and cancel out if it hangs as server may be down
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(fullTrackerURL)

	if err != nil {
		return nil, err
	}

	// read and parse body
	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)
	trackerResponse := &tracker.TrackerResponse{}
	err = bencodeparser.Read(reader, trackerResponse)
	if err != nil {
		return nil, err
	}
	return trackerResponse, nil
}

func (t TorrentClient) udpHandshakeProtocol(udpURL *url.URL, torrentfile *torrent.TorrentFile) (*tracker.TrackerResponse, error) {
	if udpURL.Scheme != "udp" {
		return nil, fmt.Errorf("invalid scheme, wanted udp got %s", udpURL.Scheme)
	}

	// send connect request and get a valid transactionID
	connectionID, err := t.sendConnectUDPReq(udpURL)

	if err != nil {
		return nil, err
	}

	fmt.Printf("passed connect request with connID of %d, attempting announce request\n", connectionID)

	// send announce request with returned connectionID for verification
	return t.sendAnnounceReq(udpURL, connectionID, torrentfile)
}

/*
announce message packet structure
Offset  Size    Name    Value
0       64-bit integer  connection_id
8       32-bit integer  action          1 // announce
12      32-bit integer  transaction_id
16      20-byte string  info_hash
36      20-byte string  peer_id
56      64-bit integer  downloaded
64      64-bit integer  left
72      64-bit integer  uploaded
80      32-bit integer  event           0 // 0: none; 1: completed; 2: started; 3: stopped
84      32-bit integer  IP address      0 // default
88      32-bit integer  key
92      32-bit integer  num_want        -1 // default
96      16-bit integer  port
98
sendAnnounceReq sends to the tracker a request to recieve valid peers
*/

func (t TorrentClient) sendAnnounceReq(udpURL *url.URL, connectionID uint64, torrentfile *torrent.TorrentFile) (*tracker.TrackerResponse, error) {
	if udpURL.Scheme != "udp" {
		return nil, fmt.Errorf("url scheme is not udp instead is %s ", udpURL.Host)
	}

	transactionID := randomUint32()

	// Build announce packet
	buf := new(bytes.Buffer)

	// connection_id
	binary.Write(buf, binary.BigEndian, connectionID)

	// action = announce (1)
	binary.Write(buf, binary.BigEndian, uint32(1))

	// transaction_id
	binary.Write(buf, binary.BigEndian, transactionID)

	// info_hash
	buf.Write(torrentfile.InfoHash[:])

	// peer_id
	buf.Write(t.peerID[:])

	// downloaded
	binary.Write(buf, binary.BigEndian, t.downloaded)

	// left
	binary.Write(buf, binary.BigEndian, t.left)

	// uploaded
	binary.Write(buf, binary.BigEndian, t.uploaded)

	// event = started (2)
	binary.Write(buf, binary.BigEndian, uint32(2))

	// IP address = 0 (default)
	binary.Write(buf, binary.BigEndian, uint32(0))

	// key (random)
	binary.Write(buf, binary.BigEndian, randomUint32())

	// num_want = -1
	binary.Write(buf, binary.BigEndian, int32(-1))

	// port
	binary.Write(buf, binary.BigEndian, uint16(t.port))

	resp, err := sendAndRecvUDP(udpURL, buf.Bytes())
	if err != nil {
		return nil, err
	}

	// response:
	// Offset      Size            Name            Value
	// 0           32-bit integer  action          1 // announce
	// 4           32-bit integer  transaction_id
	// 8           32-bit integer  interval
	// 12          32-bit integer  leechers
	// 16          32-bit integer  seeders
	// 20 + 6 * n  32-bit integer  IP address

	// now we validate the response is valid

	if len(resp) < 20 {
		// cannot be a valid response
		return nil, fmt.Errorf("response malformed : number of bytes is less than 20")
	}

	// obtain each value returned into a easy to handle variable
	action := binary.BigEndian.Uint32(resp[:4])
	respTransactionID := binary.BigEndian.Uint32(resp[4:8])

	if action != 1 {
		return nil, fmt.Errorf("response unexpected value - the value of announce in the response was not 1 (announce request response)")
	}

	if respTransactionID != transactionID {
		return nil, fmt.Errorf("transaction ID's do not match")
	}

	// valid response now

	peerBlob := resp[20:]
	if len(peerBlob)%6 != 0 {
		return nil, fmt.Errorf("length of peer blob is not a valid size 6N")
	}

	return &tracker.TrackerResponse{
		RawPeers: peerBlob,
	}, nil
}

/*
msg structure to be sent:
Offset  Size            Name            Value
0       64-bit integer  protocol_id     0x41727101980  magic constant
8       32-bit integer  action          0 // connect
12      32-bit integer  transaction_id
16

sendConnectUDPReq takes a udp url scheme and sends a connect request
to the corresponding tracker server, returns the connectionID to be presented
on each subsequent request to the tracker server from this ip:port combo until expired
*/
func (t TorrentClient) sendConnectUDPReq(udpURL *url.URL) (uint64, error) {
	// check url protocol
	if udpURL.Scheme != "udp" {
		return 0, fmt.Errorf("url scheme is not udp instead is %s ", udpURL.Host)
	}

	// build connect connectMsg
	connectMsg, transactionID := tracker.NewUDPConnectRequest()

	response, err := sendAndRecvUDP(udpURL, connectMsg.Serialize())

	if err != nil {
		return 0, err
	}

	// response will be in the shape
	// Offset  Size            Name            Value
	// 0       32-bit integer  action          0 connect
	// 4       32-bit integer  transaction_id
	// 8       64-bit integer  connection_id
	// 16

	// parse response

	if len(response) < 16 {
		return 0, fmt.Errorf("not enough bytes returned from connect to be a valid response")
	}

	responseAction := binary.BigEndian.Uint32(response[:4])
	responseTransactionID := binary.BigEndian.Uint32(response[4:8])
	connectionID := binary.BigEndian.Uint64(response[8:])

	if responseAction != 0 {
		return 0, fmt.Errorf("action does not have value 0, instead has %d", responseAction)
	}

	if transactionID != responseTransactionID {
		return 0, fmt.Errorf("transactionID does not match with generated number in request, expected %d, got %d", responseTransactionID, transactionID)
	}

	return connectionID, nil

}

// PeerHandshakeProtocol attempts to start a connection to a peer using the peer communications protocol
// this is always done via tcp or utp
func (c TorrentClient) PeerHandshakeProtocol(peer peers.Peer, infoHash [20]byte) (*net.TCPConn, error) {
	if len(peer.IP()) == 0 || peer.Port() == 0 {
		return nil, fmt.Errorf("peer is malformed - %s", peer.Address())
	}

	// build initHandshakeMsg
	initHandshakeMsg := peers.NewBitTorrentProtocolHandshake(infoHash, c.peerID)

	// attempt to connect, 5 second timeout
	conn, err := net.DialTimeout("tcp", peer.Address(), 5*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// send init msg
	conn.Write(initHandshakeMsg.SerializePeerHandshake())

	// wait for a response 5 second timeout

	readBuf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(readBuf)
	if err != nil {
		return nil, err
	}

	if n != 68 {
		return nil, fmt.Errorf("number of bytes returned is not 68, the length of the expected response instead its %d", n)
	}

	peerHandshakeResponse, err := peers.DeserializePeerHandshake([68]byte(readBuf))
	if err != nil {
		return nil, err
	}

	if peerHandshakeResponse.InfoHash != infoHash {
		return nil, fmt.Errorf("the infohash returned in the handshake are not equivilant, expected %x, got %x", infoHash, peerHandshakeResponse.InfoHash)
	}

	return nil, nil
}

// genertic func that sends a message to a url over udp and waits 5s for a response and returns it
func sendAndRecvUDP(udpURL *url.URL, msg []byte) ([]byte, error) {
	// get address
	raddr, err := net.ResolveUDPAddr("udp", udpURL.Host)
	if err != nil {
		return nil, err
	}

	// start up a socket to send the mesage
	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// listen on the socket for a response
	_, err = conn.Write(msg)
	if err != nil {
		return nil, err
	}

	// add a timeout of 5s
	err = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		return nil, err
	}

	// read response
	buf := make([]byte, 2048)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}

	return buf[:n], nil
}

func randomUint32() uint32 {
	var b [4]byte
	rand.Read(b[:])
	return binary.BigEndian.Uint32(b[:])
}
