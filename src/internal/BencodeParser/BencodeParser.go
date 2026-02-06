package bencodeparser

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
)

type BencodeInfo struct {
	Name        string     `bencode:"name"` // meaning of field dependant on type of torrent file
	Length      uint64     `bencode:"length"`
	PieceLength uint64     `bencode:"piece length"`
	Piece       [][20]byte `bencode:"pieces"` // list of sha-1 hashes with a 20 byte output
	Files       []BencodeFile
}

type BencodeFile struct {
	Path   []string `bencode:"path" json:"path"`
	Length uint64   `bencode:"length" json:"length"`
}

type Bencode struct {
	InfoHash     [20]byte    `bencode:"info hash"` // sha-1 hash of info fields raw data
	Announce     string      `bencode:"announce"`
	AnnounceList []string    `bencode: "announce list" json:"announce-list"`
	CreationDate uint64      `bencode:"creation date"`
	Info         BencodeInfo `bencode:"info"`
}

type BencodeParser struct {
	numDictsInInfoParsed int8   // number of dict value's parsed within the info key, used to understand when we are not in info anymore
	captureBytes         bool   // tells the parser when to capture bytes for info_hash calculation
	infoBytes            []byte // holds all the bytes of the info dict
	buf                  []byte
	buf_len              uint64
	cur_idx              uint64
	reader               *io.Reader
}

// == Error definitions == //

var NEGATIVE_ZERO_VALUE = fmt.Errorf("A negative zero was parsed, invalid token")
var INVALID_NEGATIVE_VALUE = fmt.Errorf("A negative number was taken for a non-negative field")
var PARSE_ERR = fmt.Errorf("Parsing error")
var EOB = fmt.Errorf("end of b.ffer error")
var EOF = fmt.Errorf("End of file error")

func makeBencodeParser(r *io.Reader) *BencodeParser {
	return &BencodeParser{
		numDictsInInfoParsed: -1,
		infoBytes:            []byte{},
		buf:                  make([]byte, 1024),
		reader:               r,
	}
}

func (b *BencodeParser) bufIdxCheckAndHandle() error {
	// Refill buffer if we've reached the end
	if b.cur_idx >= b.buf_len {

		// primarily for testing
		if b.reader == nil {
			return EOF
		}

		n, err := (*b.reader).Read(b.buf)
		if err != nil {
			return EOF
		}
		if n == 0 {
			return EOF
		}
		b.buf_len = uint64(n)
		b.cur_idx = 0
	}

	return nil
}

// gets the current token value and returns it,
// may return EOF error if index doesnt exist
// destructive process, increments current index by 1
// handles EOB
func (b *BencodeParser) consumeToken() (byte, error) {
	// Safe access because buf_len > 0 and cur_idx < buf_len

	if err := b.bufIdxCheckAndHandle(); err != nil {
		return 0x00, err
	}

	res := b.buf[b.cur_idx]

	if b.captureBytes {
		b.infoBytes = append(b.infoBytes, res)
	}

	b.cur_idx++
	return res, nil
}

// peeks at current index
// does not mutate any values
// returns EOF error
func (b *BencodeParser) peekToken() (byte, error) {
	if err := b.bufIdxCheckAndHandle(); err != nil {
		return 0x00, err
	}
	return b.buf[b.cur_idx], nil
}

func Read(reader io.Reader) (*Bencode, error) {
	if reader == nil {
		return nil, fmt.Errorf("No reader supplied")
	}

	b := makeBencodeParser(&reader)
	var torrent Bencode
	b.reader = &reader
	n, err := reader.Read(b.buf)

	if err != nil {
		return nil, err
	}
	b.buf_len = uint64(n)
	b.unmarshal(&torrent)
	return &torrent, nil
}

func flattenStringList(nested []any) []string {
	var flat []string
	for _, inner := range nested {
		// inner is actually []any
		if innerSlice, ok := inner.([]any); ok {
			for _, s := range innerSlice {
				if str, ok := s.(string); ok {
					flat = append(flat, str)
				}
			}
		}
	}
	return flat
}

func (b *BencodeParser) irToBencode(ir map[string]any, data *Bencode) error {
	// Convert IR → struct via JSON (bridge, not ideal but workable)
	marshalled, err := json.Marshal(ir)
	if err != nil {
		return fmt.Errorf("failed to marshal IR: %w", err)
	}

	if err := json.Unmarshal(marshalled, data); err != nil {
		return fmt.Errorf("failed to unmarshal IR into torrent struct: %w", err)
	}

	announcelRaw, ok := ir["announce-list"].([]any)

	if ok {
		data.AnnounceList = flattenStringList(announcelRaw)
	}

	// validating fields that must exist, and type conversions
	info, ok := ir["info"].(map[string]any)
	if !ok {
		return fmt.Errorf("info field missing or invalid")
	}

	// piece length (required, must be > 0)
	rawPieceLength, ok := info["piece length"]
	if !ok {
		return fmt.Errorf("piece length missing from info")
	}

	pieceLength, ok := asPositiveUint64(rawPieceLength)
	if !ok || pieceLength == 0 {
		return fmt.Errorf("invalid piece length")
	}

	// ---- creation date (top-level, optional but if present must be >= 0) ----
	if rawCreationDate, exists := ir["creation date"]; exists {
		creationDate, ok := asNonNegativeUint64(rawCreationDate)
		if !ok {
			return fmt.Errorf("invalid creation date")
		}
		data.CreationDate = creationDate
	}

	// ---- assign validated values ----
	data.Info.PieceLength = pieceLength
	data.InfoHash = sha1.Sum(b.infoBytes)

	return nil
}

func asNonNegativeUint64(v any) (uint64, bool) {
	switch n := v.(type) {
	case int64:
		if n < 0 {
			return 0, false
		}
		return uint64(n), true

	case int:
		if n < 0 {
			return 0, false
		}
		return uint64(n), true

	case float64:
		// JSON numbers often come through as float64
		if n < 0 || math.Trunc(n) != n {
			return 0, false
		}
		return uint64(n), true

	default:
		return 0, false
	}
}

func asPositiveUint64(v any) (uint64, bool) {
	u, ok := asNonNegativeUint64(v)
	return u, ok && u > 0
}

func prettyPrintMap(x map[string]any) {
	bc, err := json.MarshalIndent(x, "", "  ")
	if err != nil {
		// fmt.PrintLn("error:", err)
	}
	fmt.Print(string(bc))
}

func (b *BencodeParser) unmarshal(data *Bencode) error {

	IRData, err := b.parseValue()
	if err != nil {
		return fmt.Errorf("Unable to parse bencode raw data - %s", err)
	}

	// prettyPrintMap(IRData.(map[string]any))
	b.irToBencode(IRData.(map[string]any), data)

	return nil
}

func (b *BencodeParser) parseValue() (any, error) {
	// fmt.Printf("Attempting to parse key %s and index %d\n", string(b.buf[b.cur_idx]), b.cur_idx)
	if b.cur_idx >= uint64(len(b.buf)) {
		return nil, fmt.Errorf("index out of range of b.ffer")
	}

	switch string(b.buf[b.cur_idx]) {
	case "i": // int
		// fmt.PrintLn("Parsing int")
		return b.acceptInt()
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9": // string
		// fmt.PrintLn("Parsing string")
		return b.acceptString()
	case "d": // dict (map[string]any)
		// fmt.PrintLn("Parsing dict")
		return b.acceptDict()
	case "l": // list ([]any)
		// fmt.PrintLn("Parsing List")
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

	isFirst := b.numDictsInInfoParsed == 0
	if b.numDictsInInfoParsed >= 0 { // is first dict in info
		b.numDictsInInfoParsed++
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
			if _, ok := value.(map[string]any); ok && b.numDictsInInfoParsed >= 0 {
				b.numDictsInInfoParsed++
			}
			res[lastKey] = value
		}
		numParsed++
	}
	// if we are in info, decrement cause were are returning
	if isFirst {
		b.captureBytes = false
	}
	return res, nil
}

func (b *BencodeParser) acceptList() ([]any, error) {
	resList := make([]any, 0)
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
		curval, peekErr := b.peekToken()
		// consume err check
		if peekErr != nil {
			return resList, peekErr
		}
		// finished parsing
		if string(curval) == "e" {
			b.consumeToken() // get next token, cant throw error as we know e exists
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
}

func (b *BencodeParser) acceptInt() (int64, error) {
	// Expect initial 'i'
	cur, err := b.consumeToken()
	if err != nil || cur != 'i' {
		return 0, fmt.Errorf("expected 'i' at start of integer")
	}

	isNegative := false
	var num int64
	var digits int
	leadingZero := false

	for {
		cur, err = b.consumeToken()
		if err != nil {
			return 0, EOF
		}

		if cur == 'e' {
			break
		}

		// handle minus sign
		if digits == 0 && !isNegative && cur == '-' {
			isNegative = true
			continue
		}

		digit, err := isDigit(cur)
		if err != nil {
			return 0, fmt.Errorf("invalid character '%c' in integer", cur)
		}

		if digits == 0 && digit == 0 {
			leadingZero = true
		} else if leadingZero {
			// any digit after leading zero is illegal
			return 0, fmt.Errorf("invalid leading zero in integer")
		}

		num = num*10 + digit
		digits++
	}

	// no digits at all (e.g. "ie" or "i-e")
	if digits == 0 {
		return 0, fmt.Errorf("empty integer")
	}

	// reject -0
	if isNegative && num == 0 {
		return 0, NEGATIVE_ZERO_VALUE
	}

	if isNegative {
		return -num, nil
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
		b.numDictsInInfoParsed = 0 // in info key but 0 depth, waiting for dict value
		b.captureBytes = true
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
		return 0, fmt.Errorf("Unexpected token %s, expected : for end of string length or digits before this", string(curval))
	}

	return strconv.ParseUint(res, 10, 64)
}

func isDigit(c byte) (int64, error) {
	digit, err := strconv.Atoi(string(c))
	if err != nil {
		return 0, err
	}

	return int64(digit), nil
}
