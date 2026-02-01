package bencodeparser

import (
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"testing"
)

func TestParseString(t *testing.T) {
	type TestCase struct {
		testName         string
		input            string
		expectedString   string
		expectedEndIndex uint64
	}
	testcase := []TestCase{
		{"One Character String", "1:asomethingelse", "a", 3},
		{"Two Character String", "2:absomethingelse", "ab", 4},
		{"10 Character String with digit string values", "10:1234567890somethingelse", "1234567890", 13},
		{"Zero Character String", "0:sometihingelse", "", 2},
	}

	for _, tc := range testcase {
		t.Run(tc.testName, func(t *testing.T) {
			p := BencodeParser{
				buf:     []byte(tc.input),
				cur_idx: 0,
				buf_len: uint64(len([]byte(tc.input))),
			}
			gotString, _ := p.acceptString()
			if gotString != tc.expectedString {
				t.Errorf("Incorrect string value, got %s wanted %s\n", gotString, tc.expectedString)
			}
		})
	}
}

func TestGetStringLength(t *testing.T) {
	type TestCase struct {
		testName              string
		input                 string
		expectedLength        uint64
		expectedStartOfString uint64
		throwsError           bool
	}

	testcase := []TestCase{
		{"three digit number valid", "123:str", 123, 4, false},
		{"one digit valid", "9:blahblahblah", 9, 2, false},
		{"one digit invalid", "1invalid", 0, 0, true},
		{"three digit number invalid", "999blahblah", 0, 0, true},
		{"two digit invalid", "54", 0, 0, true},
	}

	for _, tc := range testcase {
		t.Run(tc.testName, func(t *testing.T) {
			p := BencodeParser{
				buf:     []byte(tc.input),
				cur_idx: 0,
				buf_len: uint64(len(tc.input)),
			}
			gotLength, gotError := p.getStringLength()

			if tc.throwsError && gotError == nil {
				t.Errorf("Expected an error didnt recieve one\n")
			}
			if tc.throwsError {
				return
			}
			if !tc.throwsError && gotError != nil {
				t.Errorf("Got an error did not expect any error :%s\n", gotError)
			}
			if tc.expectedLength != gotLength {
				t.Errorf("Invalid parsing of digits, got %d wanted %d\n", gotLength, tc.expectedLength)
			}
		})
	}
}

func TestAcceptList(t *testing.T) {
	type TestCase struct {
		testName         string
		input            string
		expected         []any
		expectedEndIndex uint64
		throwsError      bool
	}

	testcases := []TestCase{
		{
			testName:         "empty list",
			input:            "le",
			expected:         []any{},
			expectedEndIndex: 2,
			throwsError:      false,
		},
		{
			testName:         "single int",
			input:            "li32ee",
			expected:         []any{int64(32)},
			expectedEndIndex: 6,
			throwsError:      false,
		},
		{
			testName: "multiple values",
			input:    "li1e3:abce",
			expected: []any{
				int64(1),
				"abc",
			},
			expectedEndIndex: 10,
			throwsError:      false,
		},
		{
			testName: "nested list",
			input:    "lli1eee",
			expected: []any{
				[]any{
					int64(1),
				},
			},
			expectedEndIndex: 7,
			throwsError:      false,
		},
		{
			testName: "double nested list (announce-list case)",
			input:    "ll3:abceel",
			expected: []any{
				[]any{
					"abc",
				},
			},
			expectedEndIndex: 9,
			throwsError:      false,
		},
		{
			testName:    "missing end marker",
			input:       "li1e",
			expected:    nil,
			throwsError: true,
		},
		{
			testName:    "invalid token inside list",
			input:       "lxe",
			expected:    nil,
			throwsError: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.testName, func(t *testing.T) {
			p := BencodeParser{
				buf:     []byte(tc.input),
				cur_idx: 0,
				buf_len: uint64(len(tc.input)),
			}

			got, err := p.acceptList()

			if tc.throwsError {
				if err == nil {
					t.Fatalf("expected error, got none")
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
				t.Errorf(
					"wrong end index: got %d, want %d",
					p.cur_idx,
					tc.expectedEndIndex,
				)
			}
		})
	}
}
func TestParseInt(t *testing.T) {
	type TestCase struct {
		testName         string
		input            string
		expected         int64
		expectedEndIndex int64
		throwsError      bool
	}

	testcase := []TestCase{
		{"valid 32", "i32e", 32, 4, false},
		{"invalid int", "i123x42", 0, 0, true},
		{"valid 0", "i0e", 0, 3, false},
		{"invalid contains space", "i3 2e", 0, 0, true},
	}

	for _, tc := range testcase {
		t.Run(tc.testName, func(t *testing.T) {
			p := BencodeParser{
				buf:     []byte(tc.input),
				cur_idx: 0,
				buf_len: uint64(len([]byte(tc.input))),
			}

			got, err := p.acceptInt()

			if tc.throwsError && err == nil {
				t.Errorf("Expected an error did not recieve any")
			}
			if !tc.throwsError && err != nil {
				t.Errorf("Did not expect to throw an error, however did %s\n", err)
			}
			if tc.expected != got {
				t.Errorf("Wrong output got %d wanted %d\n", got, tc.expected)
			}
		})
	}
}

func TestParseList(t *testing.T) {
	type TestCase struct {
		testName string
	}
}

func TestPackage(t *testing.T) {
	type TestCase struct {
		fileName       string
		expectedOutput *BencodeTorrent
		throwsError    bool
	}
	// files with test data info
	testcase := []TestCase{
		{
			fileName: "alice.torrent",
			expectedOutput: &BencodeTorrent{
				CreationDate: 1452468725091,
				InfoHash: [20]byte{
					0x72, 0x2f, 0xe6, 0x5b, 0x2a, 0xa2, 0x6d, 0x14,
					0xf3, 0x5b, 0x4a, 0xd6, 0x27, 0xd2, 0x02, 0x36,
					0xe4, 0x81, 0xd9, 0x24,
				}, Info: BencodeInfo{
					Length:      163783,
					Name:        "alice.txt",
					PieceLength: 16384,
					Piece:       [][20]byte{}, // skip comparison for this for
				},
			},
			throwsError: false,
		},
		{
			fileName: "cosmos-laundromat.torrent",
			expectedOutput: &BencodeTorrent{
				CreationDate: 1490916617,
				Announce:     "udp://tracker.leechers-paradise.org:6969",
				InfoHash: [20]byte{
					0xc9, 0xe1, 0x57, 0x63, 0xf7, 0x22, 0xf2, 0x3e, 0x98, 0xa2,
					0x9d, 0xec, 0xdf, 0xae, 0x34, 0x1b, 0x98, 0xd5, 0x30, 0x56,
				}, Info: BencodeInfo{
					Length:      0,
					Name:        "Cosmos Laundromat",
					PieceLength: 262144,
					Piece:       [][20]byte{},
				},
			},
			throwsError: false,
		},
		{
			fileName: "big-buck-bunny.torrent",
			expectedOutput: &BencodeTorrent{
				Announce:     "udp://tracker.leechers-paradise.org:6969",
				CreationDate: 1490916601,
				InfoHash: [20]byte{
					//
					0xdd, 0x82, 0x55, 0xec, 0xdc, 0x7c, 0xa5, 0x5f,
					0xb0, 0xbb, 0xf8, 0x13, 0x23, 0xd8, 0x70, 0x62,
					0xdb, 0x1f, 0x6d, 0x1c,
				}, Info: BencodeInfo{
					Length:      0,
					Name:        "Big Buck Bunny",
					PieceLength: 262144,
					Piece:       [][20]byte{},
				},
			},
			throwsError: false,
		},
		{
			fileName: "sintel.torrent",
			expectedOutput: &BencodeTorrent{
				Announce:     "udp://tracker.leechers-paradise.org:6969",
				CreationDate: 1490916637,
				InfoHash: [20]byte{
					0x08, 0xad, 0xa5, 0xa7, 0xa6, 0x18, 0x3a, 0xae,
					0x1e, 0x09, 0xd8, 0x31, 0xdf, 0x67, 0x48, 0xd5,
					0x66, 0x09, 0x5a, 0x10,
				},
				Info: BencodeInfo{
					Length:      0,
					Name:        "Sintel",
					PieceLength: 131072,
					Piece:       [][20]byte{},
				},
			},
			throwsError: false,
		},
		{
			fileName: "wired-cd.torrent",
			expectedOutput: &BencodeTorrent{
				Announce:     "udp://tracker.leechers-paradise.org:6969",
				CreationDate: 1490916588,
				InfoHash: [20]byte{
					0xa8, 0x8f, 0xda, 0x59, 0x54, 0xe8, 0x91, 0x78,
					0xc3, 0x72, 0x71, 0x6a, 0x6a, 0x78, 0xb8, 0x18,
					0x0e, 0xd4, 0xda, 0xd3,
				},
				Info: BencodeInfo{
					Length:      0,
					Name:        "The WIRED CD - Rip. Sample. Mash. Share",
					PieceLength: 65536,
					Piece:       [][20]byte{},
				},
			},
			throwsError: false,
		},
	}
	for _, tc := range testcase {
		t.Run(tc.fileName, func(t *testing.T) {
			p := MakeBencodeParser()
			bencodeData, err := p.Read(readTestDataFile(tc.fileName))
			if !tc.throwsError && err != nil {
				t.Errorf("unexpected error thrown by Read - %s\n", err)
			}

			// we do not need to validate pieces as we can just validate the info_hash is valid
			bencodeData.Info.Piece = [][20]byte{}

			if !reflect.DeepEqual(bencodeData, tc.expectedOutput) {
				t.Errorf("got values and wanted are different\n got :\n%+v\nwanted:\n%+v\n", bencodeData, tc.expectedOutput)
			}
		})
	}
}

func readTestDataFile(filename string) io.Reader {
	testdataDir := "../testdata"
	f, err := os.Open(fmt.Sprintf("%s/%s", testdataDir, filename))
	if err != nil {
		log.Fatalln("Unable to open test file")
	}
	return f
}
