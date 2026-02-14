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
			testname: "valid initial",
			input: []byte{
				0x7F, 0x00, 0x00, 0x01, 0x1A, 0xE1,
				0xC0, 0xA8, 0x01, 0x0A, 0xC8, 0xD5,
				0x08, 0x08, 0x08, 0x08, 0x00, 0x35,
			},
			expected: []Peer{
				{
					ipv4Addr: net.IP([]byte{0x7F, 0x00, 0x00, 0x01}),
					port:     6881,
				},
				{
					ipv4Addr: net.IP([]byte{0xC0, 0xA8, 0x01, 0x0A}),
					port:     51413,
				},
				{
					ipv4Addr: net.IP([]byte{0x08, 0x08, 0x08, 0x08}),
					port:     53,
				},
			},
			throwsError: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.testname, func(t *testing.T) {
			got, gotErr := MakePeer(tc.input)

			if gotErr != nil && !tc.throwsError {
				t.Errorf("Got an unexpected error - %v", gotErr)
			}

			if reflect.DeepEqual(got, tc.expected) {
				t.Errorf("Got and wanted are not equal\nGOT:%v\nWANTED:\n%v\n", got, tc.expected)
			}
		})
	}
}
