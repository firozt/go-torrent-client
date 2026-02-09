package torrentvalidator

import (
	bencodeparser "github.com/firozt/go-torrent/src/internal/BencodeParser"
	"reflect"
	"testing"
)

func TestPackage(t *testing.T) {
	type TestCase struct {
		testname  string
		input     *bencodeparser.BencodeTorrent
		expected  Torrent
		throwsErr bool
	}

	testcases := []TestCase{
		{
			testname: "valid SFM",
			input: &bencodeparser.BencodeTorrent{
				Info: bencodeparser.BencodeInfo{
					Name:        "example.txt",
					Length:      1024,
					PieceLength: 16384,
					Piece:       "\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f\x10\x11\x12\x13\x14",
				},
				Announce:     "http://tracker.example.com/announce",
				AnnounceList: [][]any{{"http://tracker.example.com/announce"}},
				InfoHash:     [20]byte{0x10, 0x11, 0x12, 0x13}, // placeholder
				CreationDate: 1672531200,
			},
			expected: &TorrentFileSFM{
				torrentFile: torrentFile{
					Name:         "example.txt",
					Announce:     "http://tracker.example.com/announce",
					AnnounceList: []string{"http://tracker.example.com/announce"},
					PieceLength:  16384,
					Pieces: [][20]byte{
						{
							1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
							11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
						},
					},
					InfoHash:     [20]byte{0x10, 0x11, 0x12, 0x13},
					CreationDate: 1672531200,
				},
				Length: 1024,
			},
			throwsErr: false,
		},
		{
			testname: "valid MFM",
			input: &bencodeparser.BencodeTorrent{
				Info: bencodeparser.BencodeInfo{
					Name:        "music_album",
					PieceLength: 32768,
					Piece:       "\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f\x10\x11\x12\x13\x14",
					Files: []bencodeparser.BencodeFile{
						{Path: []string{"track1.mp3"}, Length: 123456},
						{Path: []string{"track2.mp3"}, Length: 654321},
					},
				},
				Announce:     "http://tracker.example.com/announce",
				AnnounceList: [][]any{{"http://tracker.example.com/announce"}, {"http://backup.tracker.com/announce"}},
				InfoHash:     [20]byte{0xaa, 0xbb, 0xcc}, // placeholder
				CreationDate: 1672531201,
			},
			expected: &TorrentFileMFM{
				torrentFile: torrentFile{
					Name:         "music_album",
					Announce:     "http://tracker.example.com/announce",
					AnnounceList: []string{"http://tracker.example.com/announce", "http://backup.tracker.com/announce"},
					PieceLength:  32768,
					Pieces: [][20]byte{
						{1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
							11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
					},
					InfoHash:     [20]byte{0xaa, 0xbb, 0xcc},
					CreationDate: 1672531201,
				},
				Files: []bencodeparser.BencodeFile{
					{Path: []string{"track1.mp3"}, Length: 123456},
					{Path: []string{"track2.mp3"}, Length: 654321},
				},
			},
			throwsErr: false,
		},
		{
			testname: "invalid pieces length",
			input: &bencodeparser.BencodeTorrent{
				Info: bencodeparser.BencodeInfo{
					Name:        "broken.txt",
					Length:      512,
					PieceLength: 16384,
					Piece:       "\x01\x02\x03", // not a multiple of 20
				},
				Announce:     "http://tracker.example.com/announce",
				AnnounceList: [][]any{{"http://tracker.example.com/announce"}},
				InfoHash:     [20]byte{0x00}, // placeholder
				CreationDate: 1672531202,
			},
			expected:  nil,
			throwsErr: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.input.Info.Name, func(t *testing.T) {
			got, err := ValidateBencodeData(tc.input)
			if err != nil && !tc.throwsErr {
				t.Errorf("Recieved an error where there was none expected - %s", err)
				return
			}
			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf("Mismatch in got and want values\nGOT:\n%+v\nWANTED:\n%+v\n", got, tc.expected)
			}
		})
	}

}
