package peers

import (
	"bytes"
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

func TestSerialize(t *testing.T) {
	type TestCase struct {
		testname string
		input    PeerHandshake
		expected [68]byte
	}

	testcases := []TestCase{
		{
			testname: "empty handshake",
			input:    PeerHandshake{},
			expected: [68]byte{},
		},

		{
			testname: "standard handshake zeros",
			input: PeerHandshake{
				StrLen:       19,
				ProtocolName: "BitTorrent protocol",
			},
			expected: [68]byte{
				19,
				'B', 'i', 't', 'T', 'o', 'r', 'r', 'e', 'n', 't', ' ',
				'p', 'r', 'o', 't', 'o', 'c', 'o', 'l',
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			},
		},

		{
			testname: "reserved bits set",
			input: PeerHandshake{
				StrLen:       19,
				ProtocolName: "BitTorrent protocol",
				Reserved:     [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
			},
			expected: [68]byte{
				19,
				'B', 'i', 't', 'T', 'o', 'r', 'r', 'e', 'n', 't', ' ',
				'p', 'r', 'o', 't', 'o', 'c', 'o', 'l',
				1, 2, 3, 4, 5, 6, 7, 8,
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			},
		},

		{
			testname: "infohash filled",
			input: PeerHandshake{
				StrLen:       19,
				ProtocolName: "BitTorrent protocol",
				InfoHash: [20]byte{
					1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
					11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
				},
			},
			expected: [68]byte{
				19,
				'B', 'i', 't', 'T', 'o', 'r', 'r', 'e', 'n', 't', ' ',
				'p', 'r', 'o', 't', 'o', 'c', 'o', 'l',
				0, 0, 0, 0, 0, 0, 0, 0,
				1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
				11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			},
		},

		{
			testname: "peer id ascii",
			input: PeerHandshake{
				StrLen:       19,
				ProtocolName: "BitTorrent protocol",
				PeerID: [20]byte{
					'-', 'G', 'O', '0', '0', '0', '1', '-',
					'1', '2', '3', '4', '5', '6', '7', '8',
					'9', '0', 'A', 'B',
				},
			},
			expected: [68]byte{
				19,
				'B', 'i', 't', 'T', 'o', 'r', 'r', 'e', 'n', 't', ' ',
				'p', 'r', 'o', 't', 'o', 'c', 'o', 'l',
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
				'-', 'G', 'O', '0', '0', '0', '1', '-',
				'1', '2', '3', '4', '5', '6', '7', '8',
				'9', '0', 'A', 'B',
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.testname, func(t *testing.T) {
			got := tc.input.SerializePeerHandshake()

			if !bytes.Equal(got, tc.expected[:]) {
				t.Errorf("expected and got are different\nGOT:%x\nWANT:%x\n", got, tc.expected)
			}
		})
	}
}

func TestDeserialize(t *testing.T) {
	type TestCase struct {
		testname  string
		input     [68]byte
		expected  *PeerHandshake
		throwsErr bool
	}

	testcases := []TestCase{
		{
			testname:  "sanity check",
			input:     [68]byte{},
			expected:  &PeerHandshake{},
			throwsErr: true, // not a peer response
		},

		{
			testname: "valid handshake minimal",
			input: [68]byte{
				19,
				'B', 'i', 't', 'T', 'o', 'r', 'r', 'e', 'n', 't', ' ',
				'p', 'r', 'o', 't', 'o', 'c', 'o', 'l',
			},
			expected: &PeerHandshake{
				StrLen:       19,
				ProtocolName: "BitTorrent protocol",
			},
			throwsErr: false,
		},

		{
			testname: "infohash present",
			input: [68]byte{
				19,
				'B', 'i', 't', 'T', 'o', 'r', 'r', 'e', 'n', 't', ' ',
				'p', 'r', 'o', 't', 'o', 'c', 'o', 'l',
				0, 0, 0, 0, 0, 0, 0, 0,
				1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
				11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
			},
			expected: &PeerHandshake{
				StrLen:       19,
				ProtocolName: "BitTorrent protocol",
				InfoHash: [20]byte{
					1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
					11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
				},
			},
			throwsErr: false,
		},

		{
			testname: "peer id present",
			input: [68]byte{
				19,
				'B', 'i', 't', 'T', 'o', 'r', 'r', 'e', 'n', 't', ' ',
				'p', 'r', 'o', 't', 'o', 'c', 'o', 'l',
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
				'-', 'G', 'O', '0', '0', '0', '1', '-',
				'1', '2', '3', '4', '5', '6', '7', '8',
				'9', '0', 'A', 'B',
			},
			expected: &PeerHandshake{
				StrLen:       19,
				ProtocolName: "BitTorrent protocol",
				PeerID: [20]byte{
					'-', 'G', 'O', '0', '0', '0', '1', '-',
					'1', '2', '3', '4', '5', '6', '7', '8',
					'9', '0', 'A', 'B',
				},
			},
			throwsErr: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.testname, func(t *testing.T) {
			got, err := DeserializePeerHandshake(tc.input)

			if tc.throwsErr && err == nil {
				t.Errorf("Expeceted an error got none")
				return
			}

			if !tc.throwsErr && err != nil {
				t.Errorf("Unexpected error - %s", err)
				return
			}

			if tc.throwsErr {
				return
			}

			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf("Got and want were different\nGOT:\n%+v\nWANT:\n%+v", got, tc.expected)
			}

		})
	}

}
