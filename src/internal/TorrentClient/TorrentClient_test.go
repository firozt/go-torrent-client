package torrentclient

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

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
			input:    "http://tracker.dmcomic.org:2710/announce",
			expected: &tracker.TrackerResponse{
				FailureReason: "no info_hash parameter supplied",
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

	for _, tc := range testcase {
		t.Run(tc.testname, func(t *testing.T) {
			client := TorrentClient{}
			u, _ := url.Parse(tc.input)
			got, err := client.handleHTTPScheme(u)

			if tc.throwsError && err == nil {
				t.Errorf("An error was expected however none were thrown")
				return
			}

			if !tc.throwsError && err != nil {
				t.Errorf("An error was thrown none expected, %v", err)
				return
			}

			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf("Got and want are not equal\nGOT:\n%v\nWANT:\n%v\n", *got, *tc.expected)
			}
		})
	}
}

func testHTTPURLSchemeSlowServer(t *testing.T) {

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
		_, serverErr := client.handleHTTPScheme(u)

		if serverErr == nil {
			t.Errorf("Expected an error did not recieve any")
		}
	})

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
