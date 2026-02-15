package peers

import (
	"net"
	"reflect"
	"testing"
)

func TestMakePeer(t *testing.T) {

	type TestCase struct {
		testname    string
		input       []byte
		expected    []Peer
		throwsError bool
	}

	testcases := []TestCase{
		{
			testname: "sanity check",
			input: []byte{
				0x7F, 0x00, 0x00, 0x01, 0x1A, 0xE1,
				0xC0, 0xA8, 0x01, 0x0A, 0xC8, 0xD5,
				0x08, 0x08, 0x08, 0x08, 0x00, 0x35,
			},
			expected: []Peer{
				{ipv4Addr: net.IP([]byte{0x7F, 0x00, 0x00, 0x01}), port: 6881},
				{ipv4Addr: net.IP([]byte{0xC0, 0xA8, 0x01, 0x0A}), port: 51413},
				{ipv4Addr: net.IP([]byte{0x08, 0x08, 0x08, 0x08}), port: 53},
			},
			throwsError: false,
		},
		{
			testname: "valid single peer",
			input: []byte{
				0xAC, 0x10, 0x00, 0x02, 0x1F, 0x90, // 172.16.0.2:8080
			},
			expected: []Peer{
				{ipv4Addr: net.IP([]byte{0xAC, 0x10, 0x00, 0x02}), port: 8080},
			},
			throwsError: false,
		},
		{
			testname: "invalid incomplete peer",
			input: []byte{
				0x7F, 0x00, 0x00, 0x01, 0x1A, // only 5 bytes instead of 6
			},
			expected:    nil,
			throwsError: true,
		},
		{
			testname:    "valid empty blob",
			input:       []byte{},
			expected:    []Peer{},
			throwsError: false,
		},
		{
			testname: "invalid length not multiple of 6",
			input: []byte{
				0x7F, 0x00, 0x00, 0x01, 0x1A, 0xE1, // one full peer
				0xC0, 0xA8, 0x01, // only 3 bytes → incomplete second peer
			},
			expected:    nil,
			throwsError: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.testname, func(t *testing.T) {
			got, gotErr := MakePeer(tc.input)

			if gotErr != nil && !tc.throwsError {
				t.Errorf("Got an unexpected error - %v", gotErr)
			}

			if gotErr == nil && tc.throwsError {
				t.Error("Error was expected, recieved none")
			}

			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf("Got and wanted are not equal\nGOT:%v\nWANTED:\n%v\n", got, tc.expected)
			}
		})
	}
}
