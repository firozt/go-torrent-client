package bencodeparser

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"testing"

	torrent "github.com/firozt/go-torrent/src/internal/Torrent"
	torrentclient "github.com/firozt/go-torrent/src/internal/TorrentClient"
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
		{"negative valid", "i-10e", -10, 5, false},
		{"negative zero invalid", "i-0e", 0, 0, true},
		{"leading zeros invalid", "i0012e", 0, 0, true},
		{"negative leading zeros invalid", "i-002e", 0, 0, true},
		{"random '-' inside invalid", "i1-2e", 0, 0, true},
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
				t.Errorf("%s: Did not expect to throw an error, however did %s\n", tc.testName, err)
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

func TestPackageTorrentData(t *testing.T) {
	type TestCase struct {
		fileName       string
		expectedOutput *torrent.RawTorrentData
		throwsError    bool
	}
	// files with test data info
	testcase := []TestCase{
		{
			fileName: "alice.torrent",
			expectedOutput: &torrent.RawTorrentData{
				CreationDate: 1452468725091,
				InfoHash: [20]byte{
					0x72, 0x2f, 0xe6, 0x5b, 0x2a, 0xa2, 0x6d, 0x14,
					0xf3, 0x5b, 0x4a, 0xd6, 0x27, 0xd2, 0x02, 0x36,
					0xe4, 0x81, 0xd9, 0x24,
				}, Info: torrent.RawTorrentInfo{
					Length:      163783,
					Name:        "alice.txt",
					PieceLength: 16384,
					Piece:       "", // skip comparison for this for

				},
			},
			throwsError: false,
		},
		{
			fileName: "cosmos-laundromat.torrent",
			expectedOutput: &torrent.RawTorrentData{
				Announce: "udp://tracker.leechers-paradise.org:6969",

				AnnounceList: [][]any{
					{"udp://tracker.leechers-paradise.org:6969"},
					{"udp://tracker.coppersurfer.tk:6969"},
					{"udp://tracker.opentrackr.org:1337"},
					{"udp://explodie.org:6969"},
					{"udp://tracker.empire-js.us:1337"},
					{"wss://tracker.btorrent.xyz"},
					{"wss://tracker.openwebtorrent.com"},
					{"wss://tracker.fastcast.nz"},
				},
				CreationDate: 1490916617,
				InfoHash: [20]byte{
					0xc9, 0xe1, 0x57, 0x63, 0xf7, 0x22, 0xf2, 0x3e, 0x98, 0xa2,
					0x9d, 0xec, 0xdf, 0xae, 0x34, 0x1b, 0x98, 0xd5, 0x30, 0x56,
				},
				Info: torrent.RawTorrentInfo{
					Length:      0, // multi-file torrent
					Name:        "Cosmos Laundromat",
					PieceLength: 262144,
					Piece:       "",
					Files: []torrent.TorrentFileField{
						{Path: []string{"Cosmos Laundromat.en.srt"}, Length: 3945},
						{Path: []string{"Cosmos Laundromat.es.srt"}, Length: 3911},
						{Path: []string{"Cosmos Laundromat.fr.srt"}, Length: 4120},
						{Path: []string{"Cosmos Laundromat.it.srt"}, Length: 3945},
						{Path: []string{"Cosmos Laundromat.mp4"}, Length: 220087570},
						{Path: []string{"poster.jpg"}, Length: 760595},
					},
				},
			},
			throwsError: false,
		},
		{
			fileName: "big-buck-bunny.torrent",
			expectedOutput: &torrent.RawTorrentData{
				Announce: "udp://tracker.leechers-paradise.org:6969",

				AnnounceList: [][]any{
					{"udp://tracker.leechers-paradise.org:6969"},
					{"udp://tracker.coppersurfer.tk:6969"},
					{"udp://tracker.opentrackr.org:1337"},
					{"udp://explodie.org:6969"},
					{"udp://tracker.empire-js.us:1337"},
					{"wss://tracker.btorrent.xyz"},
					{"wss://tracker.openwebtorrent.com"},
					{"wss://tracker.fastcast.nz"},
				},
				CreationDate: 1490916601,
				InfoHash: [20]byte{
					0xdd, 0x82, 0x55, 0xec, 0xdc, 0x7c, 0xa5, 0x5f,
					0xb0, 0xbb, 0xf8, 0x13, 0x23, 0xd8, 0x70, 0x62,
					0xdb, 0x1f, 0x6d, 0x1c,
				},
				Info: torrent.RawTorrentInfo{
					Length:      0,
					Name:        "Big Buck Bunny",
					PieceLength: 262144,
					Piece:       "",
					Files: []torrent.TorrentFileField{
						{Path: []string{"Big Buck Bunny.en.srt"}, Length: 140},
						{Path: []string{"Big Buck Bunny.mp4"}, Length: 276134947},
						{Path: []string{"poster.jpg"}, Length: 310380},
					},
				},
			},
			throwsError: false,
		},

		{
			fileName: "sintel.torrent",
			expectedOutput: &torrent.RawTorrentData{
				Announce: "udp://tracker.leechers-paradise.org:6969",

				AnnounceList: [][]any{
					{"udp://tracker.leechers-paradise.org:6969"},
					{"udp://tracker.coppersurfer.tk:6969"},
					{"udp://tracker.opentrackr.org:1337"},
					{"udp://explodie.org:6969"},
					{"udp://tracker.empire-js.us:1337"},
					{"wss://tracker.btorrent.xyz"},
					{"wss://tracker.openwebtorrent.com"},
					{"wss://tracker.fastcast.nz"},
				},
				CreationDate: 1490916637,
				InfoHash: [20]byte{
					0x08, 0xad, 0xa5, 0xa7, 0xa6, 0x18, 0x3a, 0xae,
					0x1e, 0x09, 0xd8, 0x31, 0xdf, 0x67, 0x48, 0xd5,
					0x66, 0x09, 0x5a, 0x10,
				},
				Info: torrent.RawTorrentInfo{
					Length:      0, // multi-file torrent
					Name:        "Sintel",
					PieceLength: 131072,
					Piece:       "",
					Files: []torrent.TorrentFileField{
						{Path: []string{"Sintel.de.srt"}, Length: 1652},
						{Path: []string{"Sintel.en.srt"}, Length: 1514},
						{Path: []string{"Sintel.es.srt"}, Length: 1554},
						{Path: []string{"Sintel.fr.srt"}, Length: 1618},
						{Path: []string{"Sintel.it.srt"}, Length: 1546},
						{Path: []string{"Sintel.mp4"}, Length: 129241752},
						{Path: []string{"Sintel.nl.srt"}, Length: 1537},
						{Path: []string{"Sintel.pl.srt"}, Length: 1536},
						{Path: []string{"Sintel.pt.srt"}, Length: 1551},
						{Path: []string{"Sintel.ru.srt"}, Length: 2016},
						{Path: []string{"poster.jpg"}, Length: 46115},
					},
				},
			},
			throwsError: false,
		},

		{
			fileName: "wired-cd.torrent",
			expectedOutput: &torrent.RawTorrentData{
				Announce: "udp://tracker.leechers-paradise.org:6969",

				AnnounceList: [][]any{
					{"udp://tracker.leechers-paradise.org:6969"},
					{"udp://tracker.coppersurfer.tk:6969"},
					{"udp://tracker.opentrackr.org:1337"},
					{"udp://explodie.org:6969"},
					{"udp://tracker.empire-js.us:1337"},
					{"wss://tracker.btorrent.xyz"},
					{"wss://tracker.openwebtorrent.com"},
					{"wss://tracker.fastcast.nz"},
				},
				CreationDate: 1490916588,
				InfoHash: [20]byte{
					0xa8, 0x8f, 0xda, 0x59, 0x54, 0xe8, 0x91, 0x78,
					0xc3, 0x72, 0x71, 0x6a, 0x6a, 0x78, 0xb8, 0x18,
					0x0e, 0xd4, 0xda, 0xd3,
				},
				Info: torrent.RawTorrentInfo{
					Length:      0, // multi-file torrent
					Name:        "The WIRED CD - Rip. Sample. Mash. Share",
					PieceLength: 65536,
					Piece:       "",
					Files: []torrent.TorrentFileField{
						{Path: []string{"01 - Beastie Boys - Now Get Busy.mp3"}, Length: 1964275},
						{Path: []string{"02 - David Byrne - My Fair Lady.mp3"}, Length: 3610523},
						{Path: []string{"03 - Zap Mama - Wadidyusay.mp3"}, Length: 2759377},
						{Path: []string{"04 - My Morning Jacket - One Big Holiday.mp3"}, Length: 5816537},
						{Path: []string{"05 - Spoon - Revenge!.mp3"}, Length: 2106421},
						{Path: []string{"06 - Gilberto Gil - Oslodum.mp3"}, Length: 3347550},
						{Path: []string{"07 - Dan The Automator - Relaxation Spa Treatment.mp3"}, Length: 2107577},
						{Path: []string{"08 - Thievery Corporation - Dc 3000.mp3"}, Length: 3108130},
						{Path: []string{"09 - Le Tigre - Fake French.mp3"}, Length: 3051528},
						{Path: []string{"10 - Paul Westerberg - Looking Up In Heaven.mp3"}, Length: 3270259},
						{Path: []string{"11 - Chuck D - No Meaning No (feat. Fine Arts Militia).mp3"}, Length: 3263528},
						{Path: []string{"12 - The Rapture - Sister Saviour (Blackstrobe Remix).mp3"}, Length: 6380952},
						{Path: []string{"13 - Cornelius - Wataridori 2.mp3"}, Length: 6550396},
						{Path: []string{"14 - DJ Danger Mouse - What U Sittin' On (feat. Jemini, Cee Lo And Tha Alkaholiks).mp3"}, Length: 3034692},
						{Path: []string{"15 - DJ Dolores - Oslodum 2004.mp3"}, Length: 3854611},
						{Path: []string{"16 - Matmos - Action At A Distance.mp3"}, Length: 1762120},
						{Path: []string{"README.md"}, Length: 4071},
						{Path: []string{"poster.jpg"}, Length: 78163},
					},
				},
			},
			throwsError: false,
		},
	}
	for _, tc := range testcase {
		t.Run(tc.fileName, func(t *testing.T) {
			r := readTestDataFile(tc.fileName)
			var bencodeData torrent.RawTorrentData
			err := Read(r, &bencodeData)
			if !tc.throwsError && err != nil {
				t.Errorf("unexpected error thrown by Read - %s\n", err)
			}

			// we do not need to validate pieces as we can just validate the info_hash is valid
			bencodeData.Info.Piece = ""

			if !reflect.DeepEqual(&bencodeData, tc.expectedOutput) {
				t.Errorf("got values and wanted are different\n got :\n%+v\nwanted:\n%+v\n", bencodeData, tc.expectedOutput)
			}
		})
	}
}

func TestPackageTorrentTrackerResponseData(t *testing.T) {
	type TestCase struct {
		name           string
		input          []byte // raw bencode data
		expectedOutput torrentclient.TrackerResponse
		throwsError    bool
	}

	testcases := []TestCase{
		{
			name: "simple tracker response",
			input: []byte(
				"d8:completei5e10:incompletei2e8:intervali1800e7:tracker10:tracker1235:peers12:\x7f\x00\x00\x01\x1a\xe1\xc0\xa8\x00\x02\x1a\xe1e",
			),
			expectedOutput: torrentclient.TrackerResponse{
				FailureReason: "",
				Interval:      1800,
				TrackerId:     "tracker123",
				Complete:      5,
				Incomplete:    2,
				// Peers: []torrentclient.PeerInfo{
				// 	{
				// 		PeerID:         [20]byte{1, 2, 3},
				// 		IP:             "127.0.0.1",
				// 		Port:           6881,
				// 		AmChoking:      false,
				// 		AmInterested:   false,
				// 		PeerChoking:    true,
				// 		PeerInterested: false,
				// 		LastSeen:       1700000000,
				// 	},
				// },
				Peers: "",
			},
			throwsError: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			r := bytes.NewReader(tc.input) // create a reader from bytes

			var resp torrentclient.TrackerResponse
			err := Read(r, &resp) // parse bencode from reader

			if !tc.throwsError && err != nil {
				t.Errorf("unexpected error thrown - %s", err)
				return
			}

			resp.Peers = ""
			if !reflect.DeepEqual(resp, tc.expectedOutput) {
				t.Errorf(
					"parsed TrackerResponse differs from expected\n got:\n%+v\nwanted:\n%+v\n",
					resp,
					tc.expectedOutput,
				)
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
