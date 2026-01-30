package bencodeparser

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
)

type BencodeInfo struct {
	Length     uint64     `bencode:"length"`
	Name       string     `bencode:"name"`
	PieceLenth uint64     `bencode:"piece length"`
	Piece      [][20]byte `bencode:"pieces"` // list of sha-1 hashes with a 20 byte output
}

type BencodeTorrent struct {
	InfoHash     [20]byte    `bencode:"info hash"` // sha-1 hash of info fields raw data
	Announce     string      `bencode:"announce"`
	CreationDate uint64      `bencode:"creation date"`
	Info         BencodeInfo `bencode:"info"`
}

type BencodeParser struct {
	InInfoField bool
	IR          map[string]any // intermediate representation of the BencodeData
	curParse    []byte         // holds current leaf node parsing data for change of buffers
	infoBytes   []byte         // holds all the bytes of the info dict
}

func MakeBencodeParser() *BencodeParser {
	return &BencodeParser{
		InInfoField: false,
		IR:          make(map[string]any),
		curParse:    []byte{},
		infoBytes:   []byte{},
	}
}

func (b *BencodeParser) Read(reader io.Reader) (*BencodeTorrent, error) {
	var torrent BencodeTorrent

	err := b.unmarshal(reader, &torrent)

	return &torrent, err
}

func (b *BencodeParser) IRToBencode(ir map[string]any, data *BencodeTorrent) {
	marshalled, _ := json.Marshal(ir)
	err := json.Unmarshal(marshalled, data)
	if err != nil {
		log.Fatalf("Cannot marshal unmarshaled data, should realistically never happen but linter complains")
	}

	if info, ok := ir["info"].(map[string]any); ok {
		if pieceLength, ok := info["piece length"]; ok {
			data.Info.PieceLenth = pieceLength.(uint64)
		}
	}
	data.CreationDate = ir["creation date"].(uint64)
	data.InfoHash = sha1.Sum(b.infoBytes)

	// prettyPrintMap(ir)
}

func prettyPrintMap(x map[string]any) {
	bc, err := json.MarshalIndent(x, "", "  ")
	if err != nil {
		fmt.Println("error:", err)
	}
	fmt.Print(string(bc))
}

// structure of bencode data
// objects -> d e
// integer -> i e
// strings -> {length}:{string literal}
// lists -> 	l e
func (b *BencodeParser) unmarshal(reader io.Reader, data *BencodeTorrent) error {
	buf := make([]byte, 1024)
	n, _ := reader.Read(buf)
	IRData, _, err := b.parseValue(buf[:n], 0)
	if err != nil {
		log.Fatalf("Unable to parse bencode raw data - %s", err)
	}
	b.IRToBencode(IRData.(map[string]any), data)
	// prettyPrintMap(IRData.(map[string]any))
	return nil
}

func (b *BencodeParser) parseValue(buf []byte, i uint64) (any, uint64, error) {
	if i >= uint64(len(buf)) {
		return nil, 0, fmt.Errorf("index out of range of buffer")
	}

	switch string(buf[i]) {
	case "i": // int
		return b.acceptInt(buf, i)
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9": // string
		return b.acceptString(buf, i)
	case "d": // dict (map[string]any)
		return b.acceptDict(buf, i)
	case "l": // list ([]any)
		return b.acceptList(buf, i)
	default:
		return nil, 0, fmt.Errorf("could not find a suitable accept type for %s at index %d", string(buf[i]), i)
	}
}

// TODO: Add logic to append if err = EOB error
func (b *BencodeParser) acceptDict(buf []byte, i uint64) (map[string]any, uint64, error) {
	res := make(map[string]any)

	if i >= uint64(len(buf)) || string(buf[i]) != "d" {
		return res, 0, fmt.Errorf("unable to parse dict")
	}
	numParsedFields := 0
	startIdx := i
	lastField := ""
	i++

	for i < uint64(len(buf)) {
		if string(buf[i]) == "e" {
			if b.InInfoField {
				b.InInfoField = false
				b.infoBytes = append(b.infoBytes, buf[startIdx:i+1]...) // include cur 'e' byte
			}
			break
		}
		value, newIdx, err := b.parseValue(buf, i)
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

func (b *BencodeParser) acceptList(buf []byte, i uint64) ([]any, uint64, error) {
	var resList []any
	if len(buf) < 1 || string(buf[i]) != "l" {
		return resList, 0, fmt.Errorf("invalid accept type of list has been chosen")
	}
	i++

	for i < uint64(len(buf)) {
		if string(buf[i]) == "e" {
			break
		}
		value, newIndex, err := b.parseValue(buf, i)
		if err != nil {
			return resList, 0, fmt.Errorf("unable to parse list - %s", err)
		}

		resList = append(resList, value)
		i = newIndex
	}

	return resList, i + 1, nil
}

func (b *BencodeParser) acceptInt(buf []byte, i uint64) (uint64, uint64, error) {
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
		return 0, 0, fmt.Errorf("unable to convert result into number, acceptInt func (b *BencodeTorrent)tion logic probably the cause")
	}
	return uint64(convertedRes), uint64(i + 1), nil
}

// retruns (string value, index of end of string)
func (b *BencodeParser) acceptString(buf []byte, i uint64) (string, uint64, error) {
	stringLength, strIndex, err := b.getStringLength(buf, i)
	if err != nil {
		return "", 0, fmt.Errorf("unable to parse string length - %s", err)
	}
	if strIndex+stringLength > uint64(len(buf)) {
		log.Fatalf("HAVE NOT IMPLEMENTED NEXT BUFFER HANDLING")
		// TODO: implement logic to parse next buffer also
	}

	parsedStringValue := string(buf[strIndex : strIndex+stringLength])
	if parsedStringValue == "info" {
		b.InInfoField = true
	}
	if parsedStringValue == "piece" {
	}
	return parsedStringValue, strIndex + stringLength, nil
}

// returns length of the string, index of start of string, error
func (b *BencodeParser) getStringLength(buf []byte, i uint64) (uint64, uint64, error) {
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
