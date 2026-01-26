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
	intermediateRepresentation := make(map[string]any)
	parsedNumber := 0 // number of key or fields parsed
	lastKey := ""
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if err != nil {
			break
		}

		cur := (buf[:n])
		i := 0
		for i < len(buf) {
			// check if this could be a string
			if _, err := isDigit(cur[i]); err != nil {
				text, newIdxPos := parseString(cur[i:])
				i = int(newIdxPos)
				if parsedNumber%2 == 0 { // is a key
					lastKey = text
				} else { // is a value
					intermediateRepresentation[lastKey] = text
				}
			}
			// check if this could be an int
			if string(cur[i]) == "i" {
				parseInt(cur[i:])
			}
			// // check if this could be a list
			// if c == "l" {
			// 	parseList(c)
			// }
			//
			parsedNumber++
		}

	}

	return err
}

func parseInt(buf []byte) (uint64, uint64, error) {
	// starts with an i ends with e with int between
	if len(buf) < 1 || string(buf[0]) != "i" {
		return 0, 0, fmt.Errorf("Not a valid Bencode Int")
	}
	i := 1
	res := ""
	for i < len(buf) && string(buf[i]) != "e" {
		digit, err := isDigit(buf[i])
		if err != nil {
			return 0, 0, fmt.Errorf("Invalid Bencode, cannot parse integer there is a non e terminating character %s\n", string(buf[i]))
		}
		res += strconv.FormatUint(digit, 10)
		i++
	}
	convertedRes, err := strconv.Atoi(res)
	if err != nil {
		return 0, 0, fmt.Errorf("Unable to convert result into number, parseInt function logic probably the cause\n")
	}
	return uint64(convertedRes), uint64(i + 1), nil
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
