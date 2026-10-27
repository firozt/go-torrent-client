package bencodeparser

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	torrent "github.com/firozt/go-torrent/src/internal/Torrent"
)

// newTestParser builds a parser over the given input.
func newTestParser(input string) BencodeParser {
	return BencodeParser{
		buf:     []byte(input),
		cur_idx: 0,
		buf_len: uint64(len(input)),
	}
}

// ---------------------------------------------------------------------------
// acceptString
// ---------------------------------------------------------------------------

func TestAcceptStringExtended(t *testing.T) {
	type TestCase struct {
		testName         string
		input            string
		expected         string
		expectedEndIndex uint64
		throwsError      bool
	}

	testcases := []TestCase{
		// valid
		{"empty string", "0:", "", 2, false},
		{"empty string with trailing data", "0:abc", "", 2, false},
		{"string containing colon", "3:a:b", "a:b", 5, false},
		{"string containing bencode markers", "5:i1ede", "i1ede", 7, false},
		{"string that looks like a length", "3:3:a", "3:a", 5, false},
		{"string with spaces", "5:a b c", "a b c", 7, false},
		{"string with newline", "3:a\nb", "a\nb", 5, false},
		{"binary data with null bytes", "3:\x00\xff\x01", "\x00\xff\x01", 5, false},
		{"multi-byte utf8 counts bytes not runes", "6:héllo", "héllo", 8, false},
		{"emoji is 4 bytes", "4:😀", "😀", 6, false},
		{"exactly consumes buffer", "4:abcd", "abcd", 6, false},
		{"long string (1000 chars)", "1000:" + strings.Repeat("x", 1000), strings.Repeat("x", 1000), 1005, false},

		// invalid
		{"empty input", "", "", 0, true},
		{"missing colon", "3abc", "", 0, true},
		{"missing length", ":abc", "", 0, true},
		{"length longer than buffer", "10:abc", "", 0, true},
		{"length off by one (too long)", "4:abc", "", 0, true},
		{"declared length but no data", "5:", "", 0, true},
		{"negative length", "-1:a", "", 0, true},
		{"leading zero length", "01:a", "", 0, true},
		{"non digit length", "a:b", "", 0, true},
		{"length with space", "1 :a", "", 0, true},
		{"length overflows uint64", "99999999999999999999:a", "", 0, true},
		{"only digits", "123", "", 0, true},
	}

	for _, tc := range testcases {
		t.Run(tc.testName, func(t *testing.T) {
			p := newTestParser(tc.input)
			got, err := p.acceptString()

			if tc.throwsError {
				if err == nil {
					t.Fatalf("expected error, got none (value %q)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Errorf("wrong string, got %q wanted %q", got, tc.expected)
			}
			if p.cur_idx != tc.expectedEndIndex {
				t.Errorf("wrong end index, got %d wanted %d", p.cur_idx, tc.expectedEndIndex)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// getStringLength
// ---------------------------------------------------------------------------

func TestGetStringLengthExtended(t *testing.T) {
	type TestCase struct {
		testName       string
		input          string
		expectedLength uint64
		throwsError    bool
	}

	testcases := []TestCase{
		{"zero length", "0:", 0, false},
		{"four digits", "1234:x", 1234, false},
		{"length bigger than buffer is still parsed", "50:short", 50, false},
		{"empty input", "", 0, true},
		{"just colon", ":", 0, true},
		{"letters instead of digits", "ab:cd", 0, true},
		{"negative", "-5:abcde", 0, true},
		{"plus sign", "+5:abcde", 0, true},
		{"decimal", "1.5:abc", 0, true},
		{"leading zero", "007:abcdefg", 0, true},
		{"no colon, ends at buffer end", "123", 0, true},
		{"digit then non-colon", "12x:abc", 0, true},
	}

	for _, tc := range testcases {
		t.Run(tc.testName, func(t *testing.T) {
			p := newTestParser(tc.input)
			got, err := p.getStringLength()

			if tc.throwsError {
				if err == nil {
					t.Fatalf("expected error, got none (value %d)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expectedLength {
				t.Errorf("wrong length, got %d wanted %d", got, tc.expectedLength)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// acceptInt
// ---------------------------------------------------------------------------

func TestAcceptIntExtended(t *testing.T) {
	type TestCase struct {
		testName         string
		input            string
		expected         int64
		expectedEndIndex uint64
		throwsError      bool
	}

	testcases := []TestCase{
		// valid
		{"single digit", "i7e", 7, 3, false},
		{"large positive", "i1452468725091e", 1452468725091, 15, false},
		{"negative single digit", "i-1e", -1, 4, false},
		{"max int64", "i9223372036854775807e", 9223372036854775807, 21, false},
		{"min int64", "i-9223372036854775808e", -9223372036854775808, 22, false},
		{"stops at terminator with trailing data", "i5eXYZ", 5, 3, false},
		{"followed by another int", "i1ei2e", 1, 3, false},

		// invalid
		{"empty input", "", 0, 0, true},
		{"missing prefix", "32e", 0, 0, true},
		{"missing terminator", "i32", 0, 0, true},
		{"only prefix", "i", 0, 0, true},
		{"empty body", "ie", 0, 0, true},
		{"only minus", "i-e", 0, 0, true},
		{"plus sign", "i+5e", 0, 0, true},
		{"double minus", "i--5e", 0, 0, true},
		{"decimal", "i1.5e", 0, 0, true},
		{"hex", "i0x10e", 0, 0, true},
		// {"scientific notation", "i1e5e", 0, 0, true}, TODO: comes as valid because of substr i1e
		{"trailing space", "i5 e", 0, 0, true},
		{"leading space", "i 5e", 0, 0, true},
		{"overflow int64", "i9223372036854775808e", 0, 0, true},
		{"underflow int64", "i-9223372036854775809e", 0, 0, true},
		{"way too many digits", "i" + strings.Repeat("9", 100) + "e", 0, 0, true},
		{"single leading zero before digit", "i01e", 0, 0, true},
		{"wrong prefix letter", "x5e", 0, 0, true},
		{"wrong terminator letter", "i5x", 0, 0, true},
	}

	for _, tc := range testcases {
		t.Run(tc.testName, func(t *testing.T) {
			p := newTestParser(tc.input)
			got, err := p.acceptInt()

			if tc.throwsError {
				if err == nil {
					t.Fatalf("expected error, got none (value %d)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Errorf("wrong int, got %d wanted %d", got, tc.expected)
			}
			if p.cur_idx != tc.expectedEndIndex {
				t.Errorf("wrong end index, got %d wanted %d", p.cur_idx, tc.expectedEndIndex)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// acceptList
// ---------------------------------------------------------------------------

func TestAcceptListExtended(t *testing.T) {
	type TestCase struct {
		testName         string
		input            string
		expected         []any
		expectedEndIndex uint64
		throwsError      bool
	}

	testcases := []TestCase{
		// valid
		{"list of strings", "l3:abc3:defe", []any{"abc", "def"}, 12, false},
		{"list with empty string", "l0:e", []any{""}, 4, false},
		{"list of ints", "li1ei2ei3ee", []any{int64(1), int64(2), int64(3)}, 11, false},
		{"list with negative int", "li-5ee", []any{int64(-5)}, 6, false},
		{"list with zero", "li0ee", []any{int64(0)}, 5, false},
		{"empty nested list", "llee", []any{[]any{}}, 4, false},
		{"two empty nested lists", "llelee", []any{[]any{}, []any{}}, 6, false},
		{
			"mixed types", "l4:spami42eli1eee",
			[]any{"spam", int64(42), []any{int64(1)}}, 17, false,
		},
		{
			"triple nested", "llli1eeee",
			[]any{[]any{[]any{int64(1)}}}, 9, false,
		},
		{
			"announce-list shape", "ll3:aaael3:bbbee",
			[]any{[]any{"aaa"}, []any{"bbb"}}, 16, false,
		},
		{
			"announce-list tier with multiple trackers", "ll3:aaa3:bbbee",
			[]any{[]any{"aaa", "bbb"}}, 14, false,
		},
		{"stops at end of list with trailing data", "le3:abc", []any{}, 2, false},
		{"string that looks like list terminator", "l1:ee", []any{"e"}, 5, false},
		{"string containing bencode markers", "l5:i1edee", []any{"i1ede"}, 9, false},

		// invalid
		{"empty input", "", nil, 0, true},
		{"only prefix", "l", nil, 0, true},
		{"missing prefix", "i1ee", nil, 0, true},
		{"nested list missing inner terminator", "lli1ee", nil, 0, true},
		{"unterminated after string", "l3:abc", nil, 0, true},
		{"truncated string inside list", "l5:abce", nil, 0, true},
		{"invalid int inside list (negative zero)", "li-0ee", nil, 0, true},
		{"invalid int inside list (leading zero)", "li01ee", nil, 0, true},
		{"unterminated int inside list", "li1e3:abc", nil, 0, true},
		{"unknown token inside list", "l?e", nil, 0, true},
		{"bare terminator in middle then garbage", "lei1ee", []any{}, 2, false},
	}

	for _, tc := range testcases {
		t.Run(tc.testName, func(t *testing.T) {
			p := newTestParser(tc.input)
			got, err := p.acceptList()

			if tc.throwsError {
				if err == nil {
					t.Fatalf("expected error, got none (value %#v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf("wrong output\n got:  %#v\n want: %#v", got, tc.expected)
			}
			if p.cur_idx != tc.expectedEndIndex {
				t.Errorf("wrong end index: got %d, want %d", p.cur_idx, tc.expectedEndIndex)
			}
		})
	}
}

func TestAcceptListDeeplyNestedDoesNotPanic(t *testing.T) {
	// A malicious file shouldn't be able to crash us via recursion depth.
	// We only require "no panic": either parse or return an error.
	depth := 100000
	input := strings.Repeat("l", depth) + strings.Repeat("e", depth)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("parser panicked on deep nesting: %v", r)
		}
	}()

	p := newTestParser(input)
	_, _ = p.acceptList()
}

func TestAcceptListLargeFlat(t *testing.T) {
	const n = 10000
	var sb strings.Builder
	sb.WriteString("l")
	want := make([]any, 0, n)
	for i := 0; i < n; i++ {
		sb.WriteString("i1e")
		want = append(want, int64(1))
	}
	sb.WriteString("e")

	p := newTestParser(sb.String())
	got, err := p.acceptList()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("large list mismatch (len got %d, want %d)", len(got), len(want))
	}
}

// ---------------------------------------------------------------------------
// Read: malformed / edge-case inputs (error paths only, no file needed)
// ---------------------------------------------------------------------------

func TestReadMalformedInput(t *testing.T) {
	type TestCase struct {
		testName string
		input    string
	}

	testcases := []TestCase{
		{"empty input", ""},
		{"top level int", "i5e"},
		{"top level string", "3:abc"},
		{"top level list", "l3:abce"},
		{"unterminated dict", "d"},
		{"dict missing value", "d3:fooe"},
		{"dict with int key", "di1e3:fooe"},
		{"dict unterminated after pair", "d3:foo3:bar"},
		{"truncated info dict", "d4:infod6:lengthi5e4:name1:a"},
		{"garbage", "\x00\x01\x02\x03"},
		{"plain text", "this is not a torrent file"},
	}

	for _, tc := range testcases {
		t.Run(tc.testName, func(t *testing.T) {
			var got torrent.RawTorrentData
			err := Read(bytes.NewReader([]byte(tc.input)), &got)
			if err == nil {
				t.Errorf("expected error for input %q, got none", tc.input)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Read: minimal hand-built single-file torrent
// ---------------------------------------------------------------------------

func TestReadMinimalSingleFileTorrent(t *testing.T) {
	pieces := strings.Repeat("\xab", 200) // 10 SHA-1 piece hashes

	input := "d" +
		"13:creation datei1452468725e" +
		"8:encoding5:UTF-8" +
		"4:infod" +
		"6:lengthi163783e" +
		"4:name9:alice.txt" +
		"12:piece lengthi16384e" +
		"6:pieces200:" + pieces +
		"ee"

	var got torrent.RawTorrentData
	if err := Read(bytes.NewReader([]byte(input)), &got); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Info.Name != "alice.txt" {
		t.Errorf("name: got %q, want %q", got.Info.Name, "alice.txt")
	}
	if got.Info.Length != 163783 {
		t.Errorf("length: got %d, want %d", got.Info.Length, 163783)
	}
	if got.Info.PieceLength != 16384 {
		t.Errorf("piece length: got %d, want %d", got.Info.PieceLength, 16384)
	}
	if len(got.Info.Piece) != 200 {
		t.Errorf("pieces: got %d bytes, want 200", len(got.Info.Piece))
	}
	if got.Info.Piece != pieces {
		t.Error("piece data does not match input")
	}
}

// The info hash must be the SHA1 of the *raw bytes* of the info dict, so two
// torrents that differ only in a non-info field must share an info hash.
func TestReadInfoHashIgnoresOuterFields(t *testing.T) {
	pieces := strings.Repeat("\xcd", 20)
	info := "4:info" + "d" +
		"6:length" + "i5e" +
		"4:name" + "5:a.txt" +
		"12:piece length" + "i16384e" +
		"6:pieces" + "20:" + pieces +
		"e"

	a := "d" + "8:announce" + "3:one" + info + "e"
	b := "d" + "8:announce" + "9:different" + info + "e"

	var gotA, gotB torrent.RawTorrentData
	if err := Read(bytes.NewReader([]byte(a)), &gotA); err != nil {
		t.Fatalf("a: unexpected error: %v", err)
	}
	if err := Read(bytes.NewReader([]byte(b)), &gotB); err != nil {
		t.Fatalf("b: unexpected error: %v", err)
	}

	if gotA.InfoHash != gotB.InfoHash {
		t.Errorf("info hash differs when only outer fields differ\n a: %x\n b: %x", gotA.InfoHash, gotB.InfoHash)
	}
	if gotA.InfoHash == ([20]byte{}) {
		t.Errorf("info hash was left as zero value")
	}
}

// ---------------------------------------------------------------------------
// Fuzzing: the parser must never panic on arbitrary input.
// Run with: go test -fuzz=FuzzRead -fuzztime=30s
// ---------------------------------------------------------------------------

func FuzzRead(f *testing.F) {
	seeds := []string{
		"",
		"i0e",
		"i-1e",
		"3:abc",
		"le",
		"de",
		"li1e3:abcee",
		"d3:foo3:bare",
		"d4:infod6:lengthi5e4:name1:aee",
		"ll3:abceel",
		"99999999999999999999:a",
		"i99999999999999999999e",
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var got torrent.RawTorrentData
		// We don't care whether it errors, only that it doesn't panic or hang.
		_ = Read(bytes.NewReader(data), &got)
	})
}

func FuzzAcceptList(f *testing.F) {
	for _, s := range []string{"le", "li1ee", "l3:abce", "lli1eee", "l"} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		p := newTestParser(string(data))
		_, _ = p.acceptList()
	})
}
