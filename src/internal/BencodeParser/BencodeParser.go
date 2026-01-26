package bencodeparser

import (
	"fmt"
	"io"
	"log"
	"strconv"
)

type BencodeInfo struct {
	Length     uint64     `bencode:"length"`
	Name       string     `bencode:"name"`
	PieceLenth uint64     `bencode:"piece_length"`
	Piece      [][20]byte // list of sha-1 hashes with a 20 byte output
}

type BencodeTorrent struct {
	Announce     string      `bencode:"announce"`
	CreationDate string      `bencode:"creation_date"`
	Info         BencodeInfo `bencode:"info"`
}

func Read(reader io.Reader) (*BencodeTorrent, error) {
	var torrent BencodeTorrent

	err := unmarshal(reader, &torrent)

	return &torrent, err
}

// structure of bencode data
// objects -> d e
// integer -> i e
// strings -> {length}:{string literal}
// lists -> 	l e
func unmarshal(reader io.Reader, data *BencodeTorrent) error {
	var err error
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if err != nil {
			break
		}

		cur := (buf[:n])

		for i, c := range cur {
			// check if this could be a string
			if _, err := isDigit(c); err != nil {
				text, newIdxPos := parseString(buf[i:])
				i = int(newIdxPos)
				log.Default().Print(text)
			}
			// // check if this could be an int
			// if isInt(c) {
			// 	parseInt(c)
			// }
			// // check if this could be a list
			// if c == "l" {
			// 	parseList(c)
			// }
			//
		}

	}

	return err
}

// where buf[0] is the start of the digit that represents the length
// retruns (string value, index of end of string)
func parseString(buf []byte) (string, uint64) {
	stringLength, strIndex, err := getStringLength(buf)
	if err != nil {
		log.Fatalf("unable to parse string length: %s\n", err)
	}

	if strIndex+stringLength > uint64(len(buf)) {
		log.Fatalf("HAVE NOT IMPLEMENTED NEXT BUFFER HANDLING")
		// TODO: implement logic to parse next buffer also
	}

	return string(buf[strIndex : strIndex+stringLength]), strIndex + stringLength
}

// returns length of the string, index of start of string, error
func getStringLength(buf []byte) (uint64, uint64, error) {
	res := ""
	for i := 0; i < len(buf); i++ {
		digit, err := isDigit((buf)[i])
		if err != nil {
			if string(buf[i]) != ":" {
				// invalid Bencode
				return 0, 0, fmt.Errorf("Invalid String Bencode, there is no included : after digits")
			}
			// END OF LENGTH CHECK
			break
		}
		res += strconv.FormatUint(digit, 10)
	}
	strLen, err := strconv.Atoi(res)
	if len(res) >= len(buf) || string(buf[len(res)]) != ":" {
		return 0, 0, fmt.Errorf("Invalid Bencode, there is no included : after the digits")
	}
	// +1 for the ":" character after the length and one more for the start of the string
	return uint64(strLen), uint64(len(res) + 1), err
}

func isDigit(c byte) (uint64, error) {
	digit, err := strconv.Atoi(string(c))
	if err != nil {
		return 0, err
	}

	return uint64(digit), nil
}
