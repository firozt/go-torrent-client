package torrentclient

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	torrent "github.com/firozt/go-torrent/src/internal/Torrent"
	tracker "github.com/firozt/go-torrent/src/internal/Tracker"
)

func TestHandleHTTPScheme(t *testing.T) {
	type TestCase struct {
		testname    string
		input       string // tobe converted to url obj
		expected    *tracker.TrackerResponse
		throwsError bool
	}

	testcase := []TestCase{
		{
			testname: "sanity check",
			input:    "https://tracker.moeblog.cn:443/announce",
			expected: &tracker.TrackerResponse{
				FailureReason: "",
			},
			throwsError: false,
		},
		{
			testname:    "invalid scheme",
			input:       "udp://tracker.dmcomic.org:2710/announce",
			expected:    nil,
			throwsError: true,
		},
	}
	TF := &torrent.TorrentFile{
		Name:         "ubuntu-24.04.iso",
		Announce:     []string{"udp://tracker.dmcomic.org:2710/announce", "udp://tracker.openbittorrent.com:80/announce"},
		InfoHash:     [20]byte{'T', 'E', 'S', 'T', 'I', 'N', 'G', 'H', 'A', 'S', 'H'},
		CreationDate: 1700000000,
		PieceLength:  256 * 1024,
		Pieces: [][20]byte{
			[20]byte{'P', 'I', 'E', 'C', 'E', '0', '0', '1'},
			[20]byte{'P', 'I', 'E', 'C', 'E', '0', '0', '2'},
		},
		Length: 1024 * 1024 * 1024,
		Files: []torrent.TorrentFileField{
			{
				Path:   []string{"ubuntu-24.04.iso"},
				Length: 1024 * 1024 * 1024,
			},
		},
	}
	for _, tc := range testcase[:] {
		t.Run(tc.testname, func(t *testing.T) {
			client := NewTorrentClient(1234)

			u, _ := url.Parse(tc.input)
			got, err := client.httpHandshakeProtocol(u, TF)

			if tc.throwsError && err == nil {
				t.Errorf("An error was expected however none were thrown")
				return
			}
			if !tc.throwsError && err != nil {
				t.Errorf("An error was thrown none expected, %v", err)
				return
			}

			// compare only fields we can know before making the request
			if tc.expected.FailureReason != got.FailureReason {
				t.Errorf("Got and want are not equal\nGOT:\n%+v\nWANT:\n%+v\n", *got, *tc.expected)
			}

			// if !reflect.DeepEqual(got, tc.expected) {
			// 	t.Errorf("Got and want are not equal\nGOT:\n%+v\nWANT:\n%+v\n", *got, *tc.expected)
			// }
		})
	}
}

func TestHTTPURLSchemeSlowServer(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := TorrentClient{}
	t.Run("Slow server check", func(t *testing.T) {
		u, err := url.Parse(server.URL)

		if err != nil {
			t.Errorf("DEV ERR: cannot make server - %s", err)
		}
		_, serverErr := client.httpHandshakeProtocol(u, &torrent.TorrentFile{})

		if serverErr == nil {
			t.Errorf("Expected an error did not recieve any")
		}
	})

}

func TestSendConnectUDPReq(t *testing.T) {
	type TestCase struct {
		testname    string
		input       string // tobe converted to url obj
		expected    []byte
		throwsError bool
	}

	testcase := []TestCase{
		{
			testname:    "sanity check",
			input:       "udp://tracker.opentrackr.org:1337/announce",
			throwsError: false,
		},
	}

	for _, tc := range testcase {
		t.Run(tc.testname, func(t *testing.T) {
			u, _ := url.Parse(tc.input)
			client := TorrentClient{}
			got, gotErr := client.sendConnectUDPReq(u)

			if tc.throwsError && gotErr == nil {
				t.Errorf("Expected an error however recieved none")
			}
			if !tc.throwsError && gotErr != nil {
				t.Errorf("An error was thrown none expected, %v", gotErr)
			}

			if got == 0 {
				t.Errorf("Got and want are not equal\nGOT:\n%v\nWANT:\nNON-ZERO-NUM", got)
			}
		})
	}
}

func TestUDPHandshake(t *testing.T) {
	type Input struct {
		url         string
		torrentFile torrent.TorrentFile
	}
	type TestCase struct {
		testname  string
		input     Input
		expected  tracker.TrackerResponse
		throwsErr bool
	}

	testcases := []TestCase{
		{
			testname: "sanity check",
			input: Input{
				url: "udp://tracker.opentrackr.org:1337/announce",
				torrentFile: torrent.TorrentFile{
					InfoHash: [20]byte{
						0x12, 0x34, 0x56, 0x78,
						0x9a, 0xbc, 0xde, 0xf0,
						0x11, 0x22, 0x33, 0x44,
						0x55, 0x66, 0x77, 0x88,
						0x99, 0xaa, 0xbb, 0xcc,
					},
				},
			},

			throwsErr: false,
			expected: tracker.TrackerResponse{
				FailureReason: "Your client forgot to send your torrent's info_hash. Please upgrade your client.",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.testname, func(t *testing.T) {
			client := NewTorrentClient(1234)
			got, err := client.getTrackerResponse(tc.input.url, &tc.input.torrentFile)
			if tc.throwsErr && err == nil {
				t.Errorf("Expected an error however recieved none")
			}
			if !tc.throwsErr && err != nil {
				t.Errorf("An error was thrown none expected, %v", err)
			}
			if reflect.DeepEqual(got, &tc.expected) {
				t.Errorf("got and expected are not equal\nGOT:\n%v,WANTED:\n%v", got, tc.expected)
			}

		})
	}
}

/*
http://tracker.dmcomic.org:2710/announce

curl -G http://tracker.dmcomic.org:2710/announce\
  --data-urlencode "info_hash=\x12\x34\x56\x78\x90\xab\xcd\xef\x12\x34\x56\x78\x90\xab\xcd\xef\x12\x34\x56\x78" \
  --data-urlencode "peer_id=-GO0001-123456789012" \
  --data "port=6881" \
  --data "uploaded=0" \
  --data "downloaded=0" \
  --data "left=123456789" \
  --data "compact=1" \
  --data "event=started"

*/
