// Package tracker helps to deal with response from tracker server
package tracker

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"

	peers "github.com/firozt/go-torrent/src/internal/Peers"
)

// this struct depicts the tracker response given connect + announce (or just announce via http) has been accomplished
// successfully
type TrackerResponse struct {
	FailureReason string        `json:"failure reason"`
	Interval      int64         `json:"interval"`
	TrackerID     string        `json:"tracker"`
	Complete      int64         `json:"complete"`
	Incomplete    int64         `json:"incomplete"`
	peers         *[]peers.Peer // holds parsed info from peers blob
	RawPeers      []byte        `json:"peers"`
}

// GetPeers gets peers and if non existant will generate from raw peers
// May return a raw peers does not exist error
func (t *TrackerResponse) GetPeers() (*[]peers.Peer, error) {

	if len(*t.peers) > 0 {
		return t.peers, nil
	}

	if len(t.RawPeers) == 0 {
		return nil, fmt.Errorf("peers does not exist for this variable")
	}
	val, err := peers.MakePeer([]byte(t.RawPeers))
	if err != nil {
		return nil, err
	}
	return &val, nil
}

/*
UDPConnectRequest represents the connect request via udp described in
https://www.bittorrent.org/beps/bep_0015.html, data is in the form of
Offset  Size            Name            Value
0       64-bit integer  protocol_id     0x41727101980  magic constant
8       32-bit integer  action          0 // connect
12      32-bit integer  transaction_id
16
*/
type UDPConnectRequest struct {
	ProtocolID    uint64
	Action        uint32
	TransactionID uint32
}

/*
NewUDPConnectRequest creates UDP connect message packet with randomly
generated  transactionID and returns the object and transaction ID
*/
func NewUDPConnectRequest() (*UDPConnectRequest, uint32) {
	generated := randomUint32()
	return &UDPConnectRequest{
		ProtocolID:    0x41727101980,
		Action:        uint32(0),
		TransactionID: generated,
	}, generated
}

// Serialize creates a raw byte array of the values contained
// It arranges it as ProtocolID || Action || TransactionID as per bittorrent spec
func (r UDPConnectRequest) Serialize() []byte {
	// build connect connectMsg
	msg := make([]byte, 16)
	binary.BigEndian.PutUint64(msg, r.ProtocolID)
	binary.BigEndian.PutUint32(msg[8:], r.Action)
	binary.BigEndian.PutUint32(msg[12:], r.TransactionID)

	return msg
}

// func DeserializeTrackerConnect(raw []byte) (*UDPConnectRequest, error) {
// 	if len(raw) != 16 {
// 		return nil, fmt.Errorf("input is not of a valid size (16 bytes) instead is %d", len(raw))
// 	}
//
// 	// build connect connectMsg
// 	requestTransactionID := randomUint32()
// 	connectMsg := make([]byte, 16)
// 	binary.BigEndian.PutUint64(connectMsg, 0x41727101980)
// 	binary.BigEndian.PutUint32(connectMsg[8:], 0)
// 	binary.BigEndian.PutUint32(connectMsg[12:], requestTransactionID)
//
//
//
// 	return connectMsg, nil
// }

/*
UDPAnnounceRequest represents
announce message packet stru
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
*/
type UDPAnnounceRequest struct {
	ConnectionID  int64
	TransactionID int32
	InfoHash      [20]byte
	PeerID        [20]byte
	Downloaded    int64
	Left          int64
	Uploaded      int64
	Event         int32
	IPAddress     uint32
	Key           uint32
	NumWant       int32
	Port          uint16
}

func (r UDPAnnounceRequest) Serialize() []byte {
	// TODO: implement
	return nil
}

/*
UDPConnectResponse respresents the response given from a connect request
over UDP to a tracking server documented within https://www.bittorrent.org/beps/bep_0015.html
Offset  Size            Name            Value
0       32-bit integer  action          0 connect
4       32-bit integer  transaction_id
8       64-bit integer  connection_id
16
*/
type UDPConnectResponse struct {
	Action        uint32
	TransactionID uint32
	ConnectionID  uint64
}

func (r UDPConnectResponse) Serialize() { // TODO: implement

}

// DeserializeUDPConnectResponse takes an input of raw bytes, that represents a response
// to a UDP connect request to the tracker, the shape of this is documented
// within https://www.bittorrent.org/beps/bep_0015.html
// An error will return if the input is malformed
// Note this does not validate the data given, checks of TransactionID need to be handled externally
func DeserializeUDPConnectResponse(rawInput []byte) (*UDPConnectResponse, error) {

	if len(rawInput) < 16 {
		return nil, fmt.Errorf("not enough bytes returned from connect to be a valid response")
	}

	responseAction := binary.BigEndian.Uint32(rawInput[:4])
	responseTransactionID := binary.BigEndian.Uint32(rawInput[4:8])
	connectionID := binary.BigEndian.Uint64(rawInput[8:])

	return &UDPConnectResponse{
		Action:        responseAction,
		TransactionID: responseTransactionID,
		ConnectionID:  connectionID,
	}, nil
}

// Helpers

func randomUint32() uint32 {
	var b [4]byte
	rand.Read(b[:])
	return binary.BigEndian.Uint32(b[:])
}
