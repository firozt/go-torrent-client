package torrentvalidator

import (
	"reflect"
	"testing"

	bencodeparser "github.com/firozt/go-torrent/src/internal/BencodeParser"
)

func TestAttemptParseSFM(t *testing.T) {
	type TestCase struct {
		testname  string
		input     *bencodeparser.Bencode
		expected  Torrent
		throwsErr bool
	}

	testcases := []TestCase{
		{
			testname: "valid SFM",
			input: &bencodeparser.Bencode{
				Info: bencodeparser.BencodeInfo{
					Name:        "example.txt",
					Length:      1024,
					PieceLength: 512,
					Piece: [][20]byte{
						{0x01, 0x02, 0x03},       // first piece hash (rest are zero)
						{0x0a, 0x0b, 0x0c, 0x0d}, // second piece hash
					},
				},
				Announce:     "http://tracker.example.com/announce",
				AnnounceList: []string{"http://tracker.example.com/announce"},
				InfoHash:     [20]byte{0x10, 0x11, 0x12, 0x13}, // placeholder
				CreationDate: 1672531200,
			},
			expected: &TorrentFileSFM{
				torrentFile: torrentFile{
					Name:         "example.txt",
					Announce:     "http://tracker.example.com/announce",
					AnnounceList: []string{"http://tracker.example.com/announce"},
					PieceLength:  512,
					Pieces: [][20]byte{
						{0x01, 0x02, 0x03},
						{0x0a, 0x0b, 0x0c, 0x0d},
					},
					InfoHash:     [20]byte{0x10, 0x11, 0x12, 0x13},
					CreationDate: 1672531200,
				},
				Length: 1024,
			},
			throwsErr: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.input.Info.Name, func(t *testing.T) {
			got, err := attemptParseSFM(tc.input)
			if err != nil && !tc.throwsErr {
				t.Errorf("Recieved an error where there was none expected - %s", err)
			}
			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf("Mismatch in got and want values\nGOT:\n%+v\nWANTED:\n%+v\n", got, tc.expected)
			}
		})
	}

}
