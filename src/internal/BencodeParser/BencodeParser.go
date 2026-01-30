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
	infoBytes   []byte // holds all the bytes of the info dict
	buf         []byte
	buf_len     uint64
	cur_idx     uint64
}

// == Error definitions == //

var EOB = fmt.Errorf("end of b.ffer error")

func MakeBencodeParser() *BencodeParser {
	return &BencodeParser{
		InInfoField: false,
		infoBytes:   []byte{},
		buf:         make([]byte, 1024),
	}
}

func (b *BencodeParser) Read(reader io.Reader) (*BencodeTorrent, error) {
	var torrent BencodeTorrent

	for {
		n, err := reader.Read(b.buf)
		if n > 0 {
			b.buf_len = uint64(n)
			b.unmarshal(&torrent)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}

	return &torrent, nil
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
func (b *BencodeParser) unmarshal(data *BencodeTorrent) error {

	IRData, _, err := b.parseValue()
	if err != nil {
		log.Fatalf("Unable to parse bencode raw data - %s", err)
	}

	b.IRToBencode(IRData.(map[string]any), data)
	return nil
}

func (b *BencodeParser) parseValue() (any, uint64, error) {
	if b.cur_idx >= uint64(len(b.buf)) {
		return nil, 0, fmt.Errorf("index out of range of b.ffer")
	}

	switch string(b.buf[b.cur_idx]) {
	case "i": // int
		return b.acceptInt()
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9": // string
		return b.acceptString()
	case "d": // dict (map[string]any)
		return b.acceptDict()
	case "l": // list ([]any)
		return b.acceptList()
	default:
		return nil, 0, fmt.Errorf("could not find a suitable accept type for %s at index %d", string(b.buf[b.cur_idx]), b.cur_idx)
	}
}

// TODO: Add logic to append if err = EOB error
func (b *BencodeParser) acceptDict() (map[string]any, uint64, error) {
	res := make(map[string]any)

	if b.cur_idx >= uint64(len(b.buf)) || string(b.buf[b.cur_idx]) != "d" {
		return res, 0, fmt.Errorf("unable to parse dict")
	}
	numParsedFields := 0
	startIdx := b.cur_idx
	lastField := ""
	b.cur_idx++

	for b.cur_idx < uint64(len(b.buf)) {
		if string(b.buf[b.cur_idx]) == "e" {
			if b.InInfoField {
				b.InInfoField = false
				b.infoBytes = append(b.infoBytes, b.buf[startIdx:b.cur_idx+1]...) // include cur 'e' byte
			}
			break
		}
		value, newIdx, err := b.parseValue()
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
		b.cur_idx = newIdx
		numParsedFields++
	}
	return res, b.cur_idx + 1, nil
}

func (b *BencodeParser) acceptList() ([]any, uint64, error) {
	var resList []any
	if len(b.buf) < 1 || string(b.buf[b.cur_idx]) != "l" {
		return resList, 0, fmt.Errorf("invalid accept type of list has been chosen")
	}
	b.cur_idx++

	for b.cur_idx < uint64(len(b.buf)) {
		if string(b.buf[b.cur_idx]) == "e" {
			break
		}
		value, newIndex, err := b.parseValue()
		if err != nil {
			return resList, 0, fmt.Errorf("unable to parse list - %s", err)
		}

		resList = append(resList, value)
		b.cur_idx = newIndex
	}

	return resList, b.cur_idx + 1, nil
}

func (b *BencodeParser) acceptInt() (uint64, uint64, error) {
	if b.cur_idx >= uint64(len(b.buf)) || string(b.buf[b.cur_idx]) != "i" {
		return 0, 0, fmt.Errorf("unable to parse int")
	}
	b.cur_idx++
	res := ""
	for string(b.buf[b.cur_idx]) != "e" {
		if b.cur_idx >= uint64(len(b.buf)) { // EOB, potnetially more data and may be valid, try recover

		}
		digit, err := isDigit(b.buf[b.cur_idx])
		if err != nil {
			return 0, 0, fmt.Errorf("invalid Bencode, cannot parse integer there is a non e terminating character %s", string(b.buf[b.cur_idx]))
		}
		res += strconv.FormatUint(digit, 10)
		b.cur_idx++
	}

	convertedRes, err := strconv.Atoi(res)
	if err != nil {
		return 0, 0, fmt.Errorf("unable to convert result into number, acceptInt func (b *BencodeTorrent)tion logic prob.y the cause")
	}
	return uint64(convertedRes), uint64(b.cur_idx + 1), nil
}

// retruns (string value, index of end of string)
func (b *BencodeParser) acceptString() (string, uint64, error) {
	stringLength, strIndex, err := b.getStringLength()
	if err != nil {
		return "", 0, fmt.Errorf("unable to parse string length - %s", err)
	}
	if strIndex+stringLength > uint64(len(b.buf)) {
		log.Fatalf("HAVE NOT IMPLEMENTED NEXT b.fFER HANDLING")
		// TODO: implement logic to parse next b.ffer also
	}

	parsedStringValue := string(b.buf[strIndex : strIndex+stringLength])
	if parsedStringValue == "info" {
		b.InInfoField = true
	}
	if parsedStringValue == "piece" {
	}
	return parsedStringValue, strIndex + stringLength, nil
}

// returns length of the string, index of start of string, error
func (b *BencodeParser) getStringLength() (uint64, uint64, error) {
	res := ""
	for ; b.cur_idx < uint64(len(b.buf)); b.cur_idx++ {
		digit, err := isDigit((b.buf)[b.cur_idx])
		if err != nil {
			break
		}
		res += strconv.FormatUint(digit, 10)
	}
	strLen, err := strconv.Atoi(res)
	if b.cur_idx >= uint64(len(b.buf)) || string(b.buf[b.cur_idx]) != ":" {
		return 0, 0, fmt.Errorf("invalid Bencode, there is no included ':' after the digits at index %d", b.cur_idx)
	}
	// +1 for the ":" character after the length and one more for the start of the string
	return uint64(strLen), (b.cur_idx + 1), err
}

func isDigit(c byte) (uint64, error) {
	digit, err := strconv.Atoi(string(c))
	if err != nil {
		return 0, err
	}

	return uint64(digit), nil
}
