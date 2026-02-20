// Package torrentclient that contains the client logic, high level package
package torrentclient

import (
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
		t.udpHandshakeProtocol(u)
	}

	if u.Scheme == "http" || u.Scheme == "https" {
		return t.httpHandshakeProtocol(u)
	}

	return nil, fmt.Errorf("unknown url scheme - %s", u.Scheme)
}

func (t TorrentClient) httpHandshakeProtocol(httpURL *url.URL) (*tracker.TrackerResponse, error) {
	if httpURL.Scheme != "http" && httpURL.Scheme != "https" {
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

	UDP Connect Packet Scheme

bytes	0	 2	  4 	   6	    8	    10	      12       14	16

	|--------|--------|--------|-------|--------|---------|--------|--------|

hexval   0x0000    0x0417   0x2710  0x1980 | 0x0000   0x0001  |    uint32       |

label	|---------connection_id------------|------Action------|-Transaction_id--|

	Meaning:

	connection_id ->  fixed constant defined by the protocol 64bit int
	action -> 	  action the sendee wants to accomplish (0 for connect), 32bit uint
	transaction_id -> randomly generated uint32 value to match response request

=============================================================================================
*/
func (t TorrentClient) udpHandshakeProtocol(udpURL *url.URL) (*tracker.TrackerResponse, error) {
	if udpURL.Scheme != "udp" {
		return nil, fmt.Errorf("invalid scheme, wanted udp got %s", udpURL.Scheme)
	}

	// send connect request and get a valid transactionID
	transactionID, err := t.sendConnectUDPReq(udpURL)

	transactionID++
	if err != nil {
		return nil, err
	}

	return nil, nil

}

// func (t TorrentClient) sendAnnounceReq(udpURL, *url.URL, transactionID uint32) {

// }

/*
msg structure to be sent:
Offset  Size            Name            Value
0       64-bit integer  protocol_id     0x41727101980  magic constant
8       32-bit integer  action          0 // connect
12      32-bit integer  transaction_id
16

// sendConnectUDPReq takes a udp url scheme and sends a connect request
// to the corresponding tracker server, returns the connectionID to be presented
// on each subsequent request to the tracker server from this ip:port combo until expired
*/
func (t TorrentClient) sendConnectUDPReq(udpURL *url.URL) (uint32, error) {
	// check url protocol
	if udpURL.Scheme != "udp" {
		return 0, fmt.Errorf("url scheme is not udp instead is %s ", string(udpURL.Host))
	}

	// build connect connectMsg
	requestTransactionID := randomUint32()
	connectMsg := make([]byte, 16)
	binary.BigEndian.PutUint64(connectMsg, 0x41727101980)
	binary.BigEndian.PutUint32(connectMsg[8:], 0)
	binary.BigEndian.PutUint32(connectMsg[12:], requestTransactionID)

	response, err := sendAndRecvUDP(udpURL, connectMsg)

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
	transactionID := binary.BigEndian.Uint32(response[4:8])
	connectionID := binary.BigEndian.Uint32(response[8:])

	if responseAction != 0 {
		return 0, fmt.Errorf("action does not have value 0, instead has %d", responseAction)
	}

	if transactionID != requestTransactionID {
		return 0, fmt.Errorf("transactionID does not match with genrated number in request, expected %d, got %d", transactionID, requestTransactionID)
	}

	fmt.Println(string(response))
	return connectionID, nil

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
