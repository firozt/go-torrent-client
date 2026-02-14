package peers

type PeerInfo struct {
	PeerID         [20]byte `json:"peer_id"`
	IP             string   `json:"ip"`
	Port           uint16   `json:"port"`
	AmChoking      bool     `json:"am_choking"`
	AmInterested   bool     `json:"am_interested"`
	PeerChoking    bool     `json:"peer_choking"`
	PeerInterested bool     `json:"peer_interested"`
	LastSeen       int64    `json:"last_seen"` // Unix timestamp
}
