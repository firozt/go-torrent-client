package bencodeparser

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

// maximum nesting of lists/dicts, protects against stack exhaustion on hostile input
const maxDepth = 512

type BencodeParser struct {
	depth        int    // current list/dict nesting depth (root dict == 1)
	captureBytes bool   // tells the parser when to capture bytes for info_hash calculation
	infoBytes    []byte // holds all the raw bytes of the top level "info" value
	buf          []byte
	buf_len      uint64
	cur_idx      uint64
	reader       *io.Reader
}

// == Error definitions == //

var NEGATIVE_ZERO_VALUE = fmt.Errorf("A negative zero was parsed, invalid token")
var INVALID_NEGATIVE_VALUE = fmt.Errorf("A negative number was taken for a non-negative field")
var PARSE_ERR = fmt.Errorf("Parsing error")
var EOB = fmt.Errorf("end of buffer error")
var EOF = fmt.Errorf("End of file error")

func makeBencodeParser(r *io.Reader) *BencodeParser {
	return &BencodeParser{
		infoBytes: []byte{},
		buf:       make([]byte, 1024),
		reader:    r,
	}
}

// refills the buffer when it has been fully consumed.
// returns EOF when the stream is exhausted, or the underlying read error.
func (b *BencodeParser) bufIdxCheckAndHandle() error {
	if b.cur_idx < b.buf_len {
		return nil
	}

	// primarily for testing
	if b.reader == nil || *b.reader == nil {
		return EOF
	}

	// a reader may legally return (0, nil) or (n>0, err); handle both
	for attempts := 0; attempts < 100; attempts++ {
		n, err := (*b.reader).Read(b.buf)
		if n > 0 {
			b.buf_len = uint64(n)
			b.cur_idx = 0
			return nil // any error will be returned again by the next Read
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return EOF
			}
			return err
		}
	}
	return io.ErrNoProgress
}

// gets the current token value and returns it,
// may return EOF error if index doesnt exist
// destructive process, increments current index by 1
// handles EOB
func (b *BencodeParser) consumeToken() (byte, error) {
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

// consumes exactly n bytes, copying in chunks rather than byte by byte
func (b *BencodeParser) readN(n uint64) ([]byte, error) {
	prealloc := n
	if prealloc > 64<<10 {
		prealloc = 64 << 10 // don't trust a length prefix for allocation size
	}
	out := make([]byte, 0, prealloc)

	for uint64(len(out)) < n {
		if err := b.bufIdxCheckAndHandle(); err != nil {
			return nil, err
		}
		take := b.buf_len - b.cur_idx
		if rem := n - uint64(len(out)); take > rem {
			take = rem
		}
		chunk := b.buf[b.cur_idx : b.cur_idx+take]
		out = append(out, chunk...)
		if b.captureBytes {
			b.infoBytes = append(b.infoBytes, chunk...)
		}
		b.cur_idx += take
	}
	return out, nil
}

/*
Read takes in a reader and an object to place keys into
@params
reader - reader object that represents the input stream
v - pointer to a struct (or map) in which bencode fields will be injected.
    Fields are matched by `bencode` tag, then `json` tag, then field name (case-insensitive).
    A [20]byte field named InfoHash is filled with the SHA-1 of the raw "info" dict.
@returns
returns nil reader or unparsable errors
*/
func Read(reader io.Reader, v any) error {
	if reader == nil {
		return fmt.Errorf("no reader supplied")
	}

	b := makeBencodeParser(&reader)
	return b.unmarshal(v)
}

func (b *BencodeParser) irToBencode(ir map[string]any, data any) error {
	rv := reflect.ValueOf(data).Elem()
	if err := assign(rv, ir, ""); err != nil {
		return fmt.Errorf("failed to decode into %T: %w", data, err)
	}

	// only structs can carry an InfoHash
	if rv.Kind() == reflect.Struct {
		return setInfoHash(data, b.infoBytes)
	}
	return nil
}

func setInfoHash(data any, infoBytes []byte) error {
	val := reflect.ValueOf(data)

	// Must be a pointer so we can set fields
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("data must be a pointer to a struct")
	}

	val = val.Elem() // get the struct value

	field := val.FieldByName("InfoHash")
	if !field.IsValid() || len(infoBytes) == 0 {
		return nil // nothing to set (e.g. not a torrent, or no info dict present)
	}
	if !field.CanSet() {
		return fmt.Errorf("field InfoHash cannot be set (maybe unexported)")
	}

	sum := sha1.Sum(infoBytes)

	if field.Type() == reflect.TypeOf([20]byte{}) {
		field.Set(reflect.ValueOf(sum))
		return nil
	}

	return fmt.Errorf("InfoHash field is wrong type")
}

func (b *BencodeParser) unmarshal(data any) error {
	rv := reflect.ValueOf(data)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("data must be a non-nil pointer")
	}

	first, err := b.peekToken()
	if err != nil {
		return fmt.Errorf("unable to parse bencode raw data - %w", err)
	}
	if first != 'd' {
		return fmt.Errorf("top level bencode value must be a dictionary")
	}

	IRData, err := b.parseValue()
	if err != nil {
		return fmt.Errorf("unable to parse bencode raw data - %w", err)
	}

	return b.irToBencode(IRData.(map[string]any), data)
}

func (b *BencodeParser) parseValue() (any, error) {
	c, err := b.peekToken() // also refills the buffer when needed
	if err != nil {
		return nil, err
	}

	switch {
	case c == 'i': // int
		return b.acceptInt()
	case c >= '0' && c <= '9': // string
		return b.acceptString()
	case c == 'd': // dict (map[string]any)
		return b.acceptDict()
	case c == 'l': // list ([]any)
		return b.acceptList()
	default:
		return nil, fmt.Errorf("could not find a suitable accept type for %q", c)
	}
}

func (b *BencodeParser) acceptDict() (map[string]any, error) {
	res := make(map[string]any)

	curval, consumeErr := b.consumeToken()
	if consumeErr != nil {
		return res, consumeErr
	}

	if curval != 'd' {
		return res, fmt.Errorf("Unable to parse dicitonary expected initial token 'd' however got %q\n", curval)
	}

	b.depth++
	defer func() { b.depth-- }()
	if b.depth > maxDepth {
		return res, fmt.Errorf("bencode nesting too deep")
	}
	isRoot := b.depth == 1

	for {
		c, peekErr := b.peekToken()
		if peekErr != nil {
			return res, peekErr // truncated input, don't silently accept it
		}
		if c == 'e' {
			b.consumeToken()
			return res, nil
		}

		// keys must be strings
		if c < '0' || c > '9' {
			return res, fmt.Errorf("Key is not of type string invalid bencode")
		}
		key, keyErr := b.acceptString()
		if keyErr != nil {
			return res, keyErr
		}

		// capture the raw bytes of the root dict's "info" value for the info hash
		capture := isRoot && key == "info"
		if capture {
			b.infoBytes = b.infoBytes[:0]
			b.captureBytes = true
		}

		value, valueErr := b.parseValue()

		if capture {
			b.captureBytes = false
		}
		if valueErr != nil {
			return res, valueErr
		}

		res[key] = value
	}
}

func (b *BencodeParser) acceptList() ([]any, error) {
	resList := make([]any, 0)
	curval, consumeErr := b.consumeToken()
	if consumeErr != nil {
		return resList, consumeErr
	}
	if curval != 'l' {
		return resList, fmt.Errorf("unable to parse list, initial char is not an 'l'\n")
	}

	b.depth++
	defer func() { b.depth-- }()
	if b.depth > maxDepth {
		return resList, fmt.Errorf("bencode nesting too deep")
	}

	for {
		curval, peekErr := b.peekToken()
		if peekErr != nil {
			return resList, peekErr
		}
		if curval == 'e' {
			b.consumeToken()
			break
		}

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
	if err != nil {
		return 0, err
	}
	if cur != 'i' {
		return 0, fmt.Errorf("expected 'i' at start of integer")
	}

	// collect everything up to the terminating 'e'
	raw := make([]byte, 0, 20)
	for {
		cur, err = b.consumeToken()
		if err != nil {
			return 0, err
		}
		if cur == 'e' {
			break
		}
		if len(raw) >= 20 { // longest valid int64 is 20 chars ("-9223372036854775808")
			return 0, fmt.Errorf("integer too long")
		}
		raw = append(raw, cur)
	}

	s := string(raw)
	digits := s
	isNegative := false
	if strings.HasPrefix(s, "-") {
		isNegative = true
		digits = s[1:]
	}

	// no digits at all (e.g. "ie" or "i-e")
	if digits == "" {
		return 0, fmt.Errorf("empty integer")
	}
	for i := 0; i < len(digits); i++ {
		if _, err := isDigit(digits[i]); err != nil {
			return 0, fmt.Errorf("invalid character '%c' in integer", digits[i])
		}
	}
	if len(digits) > 1 && digits[0] == '0' {
		return 0, fmt.Errorf("invalid leading zero in integer")
	}
	if isNegative && digits == "0" {
		return 0, NEGATIVE_ZERO_VALUE
	}

	// ParseInt catches overflow
	num, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid integer %q - %w", s, err)
	}
	return num, nil
}

// Strings are arbitrary bytes in bencode (e.g. the pieces field), so the raw
// bytes are kept as-is. Go strings can hold any byte sequence.
func (b *BencodeParser) acceptString() (string, error) {
	stringLength, err := b.getStringLength()
	if err != nil {
		return "", fmt.Errorf("unable to parse string length - %w", err)
	}

	data, err := b.readN(stringLength)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (b *BencodeParser) getStringLength() (uint64, error) {
	res := make([]byte, 0, 8)
	curval, consumeErr := b.consumeToken()
	for consumeErr == nil {
		if _, digitErr := isDigit(curval); digitErr != nil {
			break // non digit number
		}
		if len(res) >= 20 {
			return 0, fmt.Errorf("string length too long")
		}
		res = append(res, curval)
		curval, consumeErr = b.consumeToken()
	}

	// error with consuming next token, typically unexpected EOF
	if consumeErr != nil {
		return 0, consumeErr
	}

	// curval is not a digit
	if curval != ':' {
		return 0, fmt.Errorf("Unexpected token %q, expected : for end of string length or digits before this", curval)
	}
	if len(res) == 0 {
		return 0, fmt.Errorf("missing string length")
	}
	if len(res) > 1 && res[0] == '0' {
		return 0, fmt.Errorf("invalid leading zero in string length")
	}

	return strconv.ParseUint(string(res), 10, 64)
}

func isDigit(c byte) (int64, error) {
	if c < '0' || c > '9' {
		return 0, fmt.Errorf("%q is not a digit", c)
	}
	return int64(c - '0'), nil
}

// == IR -> user struct == //

func bencodeType(v any) string {
	switch v.(type) {
	case string:
		return "string"
	case int64:
		return "integer"
	case []any:
		return "list"
	case map[string]any:
		return "dictionary"
	}
	return fmt.Sprintf("%T", v)
}

func pathErr(path string, format string, a ...any) error {
	if path == "" {
		path = "<root>"
	}
	return fmt.Errorf("%s: %s", path, fmt.Sprintf(format, a...))
}

// assign decodes a parsed bencode value (string, int64, []any, map[string]any)
// into dst, converting directly so binary strings are never mangled.
func assign(dst reflect.Value, src any, path string) error {
	switch dst.Kind() {
	case reflect.Ptr:
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		return assign(dst.Elem(), src, path)

	case reflect.Interface:
		if dst.NumMethod() != 0 {
			return pathErr(path, "cannot decode into non-empty interface %s", dst.Type())
		}
		dst.Set(reflect.ValueOf(src))
		return nil

	case reflect.String:
		s, ok := src.(string)
		if !ok {
			return pathErr(path, "expected string, got %s", bencodeType(src))
		}
		dst.SetString(s)
		return nil

	case reflect.Bool:
		i, ok := src.(int64)
		if !ok || (i != 0 && i != 1) {
			return pathErr(path, "expected integer 0 or 1 for bool, got %s", bencodeType(src))
		}
		dst.SetBool(i == 1)
		return nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, ok := src.(int64)
		if !ok {
			return pathErr(path, "expected integer, got %s", bencodeType(src))
		}
		if dst.OverflowInt(i) {
			return pathErr(path, "integer %d overflows %s", i, dst.Type())
		}
		dst.SetInt(i)
		return nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, ok := src.(int64)
		if !ok {
			return pathErr(path, "expected integer, got %s", bencodeType(src))
		}
		if i < 0 {
			return fmt.Errorf("%s: %w", path, INVALID_NEGATIVE_VALUE)
		}
		if dst.OverflowUint(uint64(i)) {
			return pathErr(path, "integer %d overflows %s", i, dst.Type())
		}
		dst.SetUint(uint64(i))
		return nil

	case reflect.Slice:
		if dst.Type().Elem().Kind() == reflect.Uint8 {
			s, ok := src.(string)
			if !ok {
				return pathErr(path, "expected string for byte slice, got %s", bencodeType(src))
			}
			dst.SetBytes([]byte(s))
			return nil
		}
		list, ok := src.([]any)
		if !ok {
			return pathErr(path, "expected list, got %s", bencodeType(src))
		}
		out := reflect.MakeSlice(dst.Type(), len(list), len(list))
		for i := range list {
			if err := assign(out.Index(i), list[i], fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
		dst.Set(out)
		return nil

	case reflect.Array:
		if dst.Type().Elem().Kind() == reflect.Uint8 {
			s, ok := src.(string)
			if !ok {
				return pathErr(path, "expected string for byte array, got %s", bencodeType(src))
			}
			if len(s) != dst.Len() {
				return pathErr(path, "expected %d bytes, got %d", dst.Len(), len(s))
			}
			reflect.Copy(dst, reflect.ValueOf(s))
			return nil
		}
		list, ok := src.([]any)
		if !ok {
			return pathErr(path, "expected list, got %s", bencodeType(src))
		}
		if len(list) != dst.Len() {
			return pathErr(path, "expected %d elements, got %d", dst.Len(), len(list))
		}
		for i := range list {
			if err := assign(dst.Index(i), list[i], fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
		return nil

	case reflect.Map:
		if dst.Type().Key().Kind() != reflect.String {
			return pathErr(path, "map keys must be strings, got %s", dst.Type().Key())
		}
		m, ok := src.(map[string]any)
		if !ok {
			return pathErr(path, "expected dictionary, got %s", bencodeType(src))
		}
		if dst.IsNil() {
			dst.Set(reflect.MakeMapWithSize(dst.Type(), len(m)))
		}
		for k, v := range m {
			elem := reflect.New(dst.Type().Elem()).Elem()
			if err := assign(elem, v, path+"."+k); err != nil {
				return err
			}
			dst.SetMapIndex(reflect.ValueOf(k).Convert(dst.Type().Key()), elem)
		}
		return nil

	case reflect.Struct:
		m, ok := src.(map[string]any)
		if !ok {
			return pathErr(path, "expected dictionary, got %s", bencodeType(src))
		}
		exact, folded := structFieldIndex(dst.Type())
		for key, v := range m {
			idx, found := exact[key]
			if !found {
				idx, found = folded[strings.ToLower(key)]
			}
			if !found {
				continue // unknown keys are ignored
			}
			if err := assign(dst.Field(idx), v, path+"."+key); err != nil {
				return err
			}
		}
		return nil
	}

	return pathErr(path, "unsupported destination type %s", dst.Type())
}

// maps bencode keys to struct field indexes using the `bencode` tag, then the
// `json` tag, then the field name.
func structFieldIndex(t reflect.Type) (exact map[string]int, folded map[string]int) {
	exact = make(map[string]int)
	folded = make(map[string]int)
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" { // unexported
			continue
		}
		name := sf.Name
		tag := sf.Tag.Get("bencode")
		if tag == "" {
			tag = sf.Tag.Get("json")
		}
		if tag != "" {
			tag = strings.Split(tag, ",")[0]
			if tag == "-" {
				continue
			}
			if tag != "" {
				name = tag
			}
		}
		exact[name] = i
		if _, dup := folded[strings.ToLower(name)]; !dup {
			folded[strings.ToLower(name)] = i
		}
	}
	return exact, folded
}
