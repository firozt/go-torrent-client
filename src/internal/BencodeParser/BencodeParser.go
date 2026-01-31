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
	infoDictDepth int8
	infoBytes     []byte // holds all the bytes of the info dict
	buf           []byte
	buf_len       uint64
	cur_idx       uint64
	reader        *io.Reader
}

// == Error definitions == //

var PARSE_ERR = fmt.Errorf("Parsing error")
var EOB = fmt.Errorf("end of b.ffer error")
var EOF = fmt.Errorf("End of file error")

func MakeBencodeParser() *BencodeParser {
	return &BencodeParser{
		infoDictDepth: -1,
		infoBytes:     []byte{},
		buf:           make([]byte, 1024),
	}
}

// gets the current token value and returns it,
// may return EOF error if index doesnt exist
// destructive process, increments current index by 1
// handles EOB
func (b *BencodeParser) consumeToken() (byte, error) {
	// Refill buffer if we've reached the end
	if b.cur_idx >= b.buf_len {

		// primarily for testing
		if b.reader == nil {
			return 0x00, EOF
		}

		n, err := (*b.reader).Read(b.buf)
		if err != nil {
			return 0x00, EOF
		}
		if n == 0 {
			// Prevent panic if reader returned 0 bytes but no error
			return 0x00, EOF
		}
		b.buf_len = uint64(n)
		b.cur_idx = 0
	}

	// Safe access because buf_len > 0 and cur_idx < buf_len

	res := b.buf[b.cur_idx]

	if b.infoDictDepth >= 0 {
		b.infoBytes = append(b.infoBytes, res)
	}

	b.cur_idx++
	return res, nil
}

// peeks at current index
// does not mutate any values
// returns EOF error
func (b *BencodeParser) peekToken() (byte, error) {
	// TODO: handle EOB
	return b.buf[b.cur_idx], nil
}

func (b *BencodeParser) Read(reader io.Reader) (*BencodeTorrent, error) {
	var torrent BencodeTorrent
	b.reader = &reader
	n, err := reader.Read(b.buf)

	if err != nil {
		return nil, err
	}
	b.buf_len = uint64(n)
	b.unmarshal(&torrent)

	// for {
	// 	n, err := reader.Read(b.buf)
	// 	if n > 0 {
	// 		b.buf_len = uint64(n)
	// 		b.unmarshal(&torrent)
	// 	}
	// 	if err != nil {
	// 		if err == io.EOF {
	// 			break
	// 		}
	// 		return nil, err
	// 	}
	// }

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

	IRData, err := b.parseValue()
	if err != nil {
		log.Fatalf("Unable to parse bencode raw data - %s", err)
	}

	b.IRToBencode(IRData.(map[string]any), data)
	return nil
}

func (b *BencodeParser) parseValue() (any, error) {
	fmt.Printf("Attempting to parse key %s and index %d\n", string(b.buf[b.cur_idx]), b.cur_idx)
	if b.cur_idx >= uint64(len(b.buf)) {
		return nil, fmt.Errorf("index out of range of b.ffer")
	}

	switch string(b.buf[b.cur_idx]) {
	case "i": // int
		fmt.Println("Parsing int")
		return b.acceptInt()
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9": // string
		fmt.Println("Parsing string")
		return b.acceptString()
	case "d": // dict (map[string]any)
		fmt.Println("Parsing dict")
		return b.acceptDict()
	case "l": // list ([]any)
		fmt.Println("Parsing List")
		return b.acceptList()
	default:
		return nil, fmt.Errorf("could not find a suitable accept type for %s at index %d", string(b.buf[b.cur_idx]), b.cur_idx)
	}
}

func (b *BencodeParser) acceptDict() (map[string]any, error) {
	res := make(map[string]any)

	curval, consumeErr := b.consumeToken()
	// consume err check
	if consumeErr != nil {
		return res, consumeErr
	}

	// check if valid call of acceptDict()
	if string(curval) != "d" {
		return res, fmt.Errorf("Unable to parse dicitonary expected initial token 'd' however got %s\n", string(curval))
	}

	// can now parse dict bytes
	numParsed := 0
	lastKey := ""
	for {
		curval, peekError := b.peekToken()
		if peekError != nil {
			return res, nil

		}
		if string(curval) == "e" {
			b.consumeToken() // get ready for next parse
			if b.infoDictDepth >= 0 {
				b.infoDictDepth--
			}
			break
		}

		value, valueErr := b.parseValue()

		// check value err
		if valueErr != nil {
			return res, valueErr
		}

		// is a string
		if numParsed%2 == 0 {
			// must be of type string
			s, ok := value.(string)

			if !ok {
				return res, fmt.Errorf("Key is not of type string invalid bencode")
			}
			lastKey = s

		} else { // is a value
			// check if inside info, and check if its a dict if so increment depth
			if _, ok := value.(map[string]any); ok && b.infoDictDepth >= 0 {
				b.infoDictDepth++
			}
			res[lastKey] = value
		}
		numParsed++
	}
	return res, nil
}

func (b *BencodeParser) acceptList() ([]any, error) {
	var resList []any
	curval, consumeErr := b.consumeToken()
	// check consume error
	if consumeErr != nil {
		return resList, consumeErr
	}
	// check if this is a valid call of acceptList()
	if string(curval) != "l" {
		return resList, fmt.Errorf("unable to parse list, initial char is not an 'l'\n")
	}
	// start parsing value bytes
	for {
		curval, consumeErr = b.consumeToken()
		// consume err check
		if consumeErr != nil {
			return resList, consumeErr
		}
		// finished parsing
		if string(curval) == "e" {
			break
		}
		// valid parse value
		value, valueErr := b.parseValue()

		if valueErr != nil {
			return resList, valueErr
		}

		resList = append(resList, value)
	}

	return resList, nil

	// var resList []any
	// if len(b.buf) < 1 || string(b.buf[b.cur_idx]) != "l" {
	// 	return resList, fmt.Errorf("invalid accept type of list has been chosen")
	// }
	// b.cur_idx++
	//
	// for b.cur_idx < uint64(len(b.buf)) {
	// 	if string(b.buf[b.cur_idx]) == "e" {
	// 		break
	// 	}
	// 	value, err := b.parseValue()
	// 	if err != nil {
	// 		return resList, fmt.Errorf("unable to parse list - %s", err)
	// 	}
	//
	// 	resList = append(resList, value)
	// }
	//
	// return resList, nil
}

func (b *BencodeParser) acceptInt() (uint64, error) {
	// Expect initial 'i'
	cur, err := b.consumeToken()
	if err != nil || cur != 'i' {
		return 0, fmt.Errorf("expected 'i' at start of integer")
	}

	// break if EOF, recieve 'e' or unparasable integer digit
	var num uint64
	for {
		cur, err = b.consumeToken()

		if err != nil {
			return 0, EOF
		}
		if cur == 'e' {
			break // end of integer
		}

		digit, err := isDigit(cur)
		if err != nil {
			return 0, fmt.Errorf("invalid character '%c' in integer", cur)
		}

		num = num*10 + digit
	}

	return num, nil
}

// retruns (string value, index of end of string)
func (b *BencodeParser) acceptString() (string, error) {
	// get string length parsed
	stringLength, err := b.getStringLength()
	if err != nil {
		return "", fmt.Errorf("unable to parse string length - %s", err)
	}

	// cur token should now be start of the string
	res := ""

	// we know how long to scan for, only error can be EOF
	for i := 0; i < int(stringLength); i++ {
		curval, consumeErr := b.consumeToken()
		if consumeErr != nil {
			return "", consumeErr
		}
		res += string(curval)
	}

	if res == "info" {
		b.infoDictDepth = 0 // in info key but 0 depth, waiting for dict value
	}
	// cur token should now be start of next item
	return res, nil
}

func (b *BencodeParser) getStringLength() (uint64, error) {
	res := ""
	curval, consumeErr := b.consumeToken()
	for consumeErr == nil {
		_, digitErr := isDigit(curval)
		if digitErr != nil {
			break // non digit number
		}
		res += string(curval)
		curval, consumeErr = b.consumeToken()
	}

	// error with consuming next token, typically unexpected EOF
	if consumeErr != nil {
		return 0, consumeErr
	}

	// curval is not a digit
	if string(curval) != ":" {
		return 0, fmt.Errorf("Unexpected token %s, expected : for end of string length", string(curval))
	}

	return strconv.ParseUint(res, 10, 64)
}

func isDigit(c byte) (uint64, error) {
	digit, err := strconv.Atoi(string(c))
	if err != nil {
		return 0, err
	}

	return uint64(digit), nil
}
