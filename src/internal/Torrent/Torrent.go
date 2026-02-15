// Package torrent cotains representations of a .torrent file, contains methods that help with the bittorrent protocal
package torrent

import (
	"fmt"
	"net/url"
	"strconv"
)

// ============ Struct Defs  ============ //

// TorrentFile flattened torrentfile struct with better typing, enforcing field values types
type TorrentFile struct {
	Name         string
	Announce     []string
	InfoHash     [20]byte
	CreationDate uint64
	PieceLength  uint64
	Pieces       [][20]byte
	Length       uint64
	Files        []TorrentFileField
}

// ============ Raw Data Structs  ============ //

// RawTorrentInfo raw direct representation of the bencode struct
type RawTorrentInfo struct {
	Name        string             `bencode:"name" json:"name"`
	Length      int64              `bencode:"length" json:"length"`
	PieceLength int64              `bencode:"piece length" json:"piece length"`
	Piece       string             `bencode:"pieces" json:"pieces"`
	Files       []TorrentFileField `bencode:"files" json:"files"`
}

type TorrentFileField struct {
	Path   []string `bencode:"path" json:"path"`
	Length int64    `bencode:"length" json:"length"`
}

type RawTorrentData struct {
	InfoHash     [20]byte       `bencode:"info hash" json:"info_hash"`
	Announce     string         `bencode:"announce" json:"announce"`
	AnnounceList [][]any        `bencode:"announce list" json:"announce-list"`
	CreationDate int64          `bencode:"creation date" json:"creation date"`
	Info         RawTorrentInfo `bencode:"info" json:"info"`
}

// ============ Methods  ============ //

func (t *TorrentFile) IsMultiFile() bool {
	if len(t.Files) > 0 && len(t.Files[0].Path) != 0 {
		return true
	}

	if t.Length != 0 {
		return false
	}

	panic("Torrentfile is neither SFM or MFM")
}

// BuildTrackerURL builds a tracker url given an announce url string
func (t TorrentFile) BuildTrackerURL(announce string, peerID string, port uint16) (string, error) {
	_, err := url.Parse(announce)

	if err != nil {
		return "", fmt.Errorf("invalid url give %s, ", announce)
	}

	return announce + "?" + t.buildParams(peerID, port), nil

}

func (t TorrentFile) BuildAllTrackerURL(peerID string, port uint16) []string {

	var trackerUrls []string
	params := t.buildParams(peerID, port)

	for _, announceURL := range t.Announce {
		_, err := url.Parse(announceURL)
		if err != nil {
			continue
		}
		trackerUrls = append(trackerUrls, announceURL+"?"+params)
	}

	return trackerUrls
}

func (t TorrentFile) buildParams(peerID string, port uint16) string {
	params := url.Values{
		"info_hash":  []string{string(t.InfoHash[:])},
		"peer_id":    []string{peerID},
		"port":       []string{strconv.Itoa(int(port))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.FormatUint(t.Length, 10)},
	}
	return params.Encode()

}
