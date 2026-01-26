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
	buf := make([]byte, 1024)
	n, _ := reader.Read(buf)
	IRData, _, err := parseValue(buf[:n], 0)
	if err != nil {
		log.Fatalf("Unable to parse bencode raw data - %s", err)
	}
	fmt.Println(IRData)
	return nil
}

func parseValue(buf []byte, i uint64) (any, uint64, error) {
	if i >= uint64(len(buf)) {
		return nil, 0, fmt.Errorf("index out of range of buffer")
	}

	switch string(buf[i]) {
	case "i": // int
		return parseInt(buf, i)
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9": // string
		return parseString(buf, i)
	case "d": // dict (map[string]any)
		return parseDict(buf, i)
	case "l": // list ([]any)
		return parseList(buf, i)
	default:
		return nil, 0, fmt.Errorf("could not find a suitable accept type for %s at index %d", string(buf[i]), i)
	}
}

func parseDict(buf []byte, i uint64) (map[string]any, uint64, error) {
	res := make(map[string]any)

	if i >= uint64(len(buf)) || string(buf[i]) != "d" {
		return res, 0, fmt.Errorf("unable to parse dict")
	}
	i++

	numParsedFields := 0
	lastField := ""
	for i < uint64(len(buf)) {
		if string(buf[i]) == "e" {
			break
		}
		value, newIdx, err := parseValue(buf, i)
		if err != nil {
			return map[string]any{}, 0, fmt.Errorf("unable to parse dict - %s", err)
		}
		if numParsedFields%2 == 0 { // is a key
			strVal, ok := value.(string)
			if !ok {
				return res, 0, fmt.Errorf("unable to parse bencode, key is not of type string")
			}
			lastField = strVal
		} else { // is a value
			res[lastField] = value
		}
		i = newIdx
		numParsedFields++
	}
	return res, i + 1, nil
}

func parseList(buf []byte, i uint64) ([]any, uint64, error) {
	var resList []any
	if len(buf) < 1 || string(buf[i]) != "l" {
		return resList, 0, fmt.Errorf("invalid accept type of list has been chosen")
	}
	i++

	for i < uint64(len(buf)) {
		if string(buf[i]) == "e" {
			break
		}
		value, newIndex, err := parseValue(buf, i)
		if err != nil {
			return resList, 0, fmt.Errorf("unable to parse list - %s", err)
		}

		resList = append(resList, value)
		i = newIndex
	}

	return resList, i + 1, nil
}

func parseInt(buf []byte, i uint64) (uint64, uint64, error) {
	if i >= uint64(len(buf)) || string(buf[i]) != "i" {
		return 0, 0, fmt.Errorf("unable to parse int")
	}
	i++
	res := ""
	for i < uint64(len(buf)) && string(buf[i]) != "e" {
		digit, err := isDigit(buf[i])
		if err != nil {
			return 0, 0, fmt.Errorf("invalid Bencode, cannot parse integer there is a non e terminating character %s", string(buf[i]))
		}
		res += strconv.FormatUint(digit, 10)
		i++
	}
	convertedRes, err := strconv.Atoi(res)
	if err != nil {
		return 0, 0, fmt.Errorf("unable to convert result into number, parseInt function logic probably the cause")
	}
	return uint64(convertedRes), uint64(i + 1), nil
}

// where buf[0] is the start of the digit that represents the length
// retruns (string value, index of end of string)
func parseString(buf []byte, i uint64) (string, uint64, error) {
	stringLength, strIndex, err := getStringLength(buf, i)
	if err != nil {
		return "", 0, fmt.Errorf("unable to parse string length - %s", err)
	}
	if strIndex+stringLength > uint64(len(buf)) {
		log.Fatalf("HAVE NOT IMPLEMENTED NEXT BUFFER HANDLING")
		// TODO: implement logic to parse next buffer also
	}

	return string(buf[strIndex : strIndex+stringLength]), strIndex + stringLength, nil
}

// returns length of the string, index of start of string, error
func getStringLength(buf []byte, i uint64) (uint64, uint64, error) {
	res := ""
	for ; i < uint64(len(buf)); i++ {
		digit, err := isDigit((buf)[i])
		if err != nil {
			break
		}
		res += strconv.FormatUint(digit, 10)
	}
	strLen, err := strconv.Atoi(res)
	if i >= uint64(len(buf)) || string(buf[i]) != ":" {
		return 0, 0, fmt.Errorf("invalid Bencode, there is no included ':' after the digits at index %d", i)
	}
	// +1 for the ":" character after the length and one more for the start of the string
	return uint64(strLen), (i + 1), err
}

func isDigit(c byte) (uint64, error) {
	digit, err := strconv.Atoi(string(c))
	if err != nil {
		return 0, err
	}

	return uint64(digit), nil
}
