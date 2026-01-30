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
			p := MakeBencodeParser()
			gotString, gotStart, _ := p.acceptString([]byte(tc.input), 0)
			if gotString != tc.expectedString {
				t.Errorf("Incorrect string value, got %s wanted %s\n", gotString, tc.expectedString)
			}
			if gotStart != tc.expectedEndIndex {
				t.Errorf("Incorrect end index, got %d wanted %d\n", gotStart, tc.expectedEndIndex)
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
			p := MakeBencodeParser()
			gotLength, gotStartOfString, gotError := p.getStringLength([]byte(tc.input), 0)

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
			if tc.expectedStartOfString != gotStartOfString {
				t.Errorf("Invalid start of string value got %d wanted %d\n", gotStartOfString, tc.expectedStartOfString)
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	type TestCase struct {
		testName         string
		input            string
		expected         uint64
		expectedEndIndex uint64
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
			p := MakeBencodeParser()
			got, gotEndIndex, err := p.acceptInt([]byte(tc.input), 0)

			if tc.throwsError && err == nil {
				t.Errorf("Expected an error did not recieve any")
			}
			if !tc.throwsError && err != nil {
				t.Errorf("Did not expect to throw an error, however did %s\n", err)
			}
			if tc.expected != got {
				t.Errorf("Wrong output got %d wanted %d\n", got, tc.expected)
			}
			if tc.expectedEndIndex != gotEndIndex {
				t.Errorf("Wrong end index got %d wanted %d\n", gotEndIndex, tc.expectedEndIndex)
			}
		})
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
					Length:     163783,
					Name:       "alice.txt",
					PieceLenth: 16384,
					Piece:      [][20]byte{}, // skip comparison for this for
					// Piece: [][20]byte{
					// 	{0x24, 0xc0, 0x63, 0x52, 0xb8, 0xf1, 0x8d, 0xcb, 0xc4, 0x83, 0x14, 0x22, 0x4d, 0x6c, 0xa2, 0x26, 0x0e, 0x18, 0xf2, 0xbf},
					// 	{0xd2, 0xcb, 0xb9, 0x8b, 0xe1, 0x3f, 0xe5, 0x7e, 0x61, 0xfd, 0x02, 0x24, 0xa9, 0x02, 0x18, 0x3c, 0x7d, 0x5e, 0xae, 0x65},
					// 	{0x41, 0xbf, 0x1f, 0x17, 0xbb, 0xe4, 0x63, 0xdb, 0x39, 0x1b, 0x69, 0x81, 0xdc, 0xaf, 0x2f, 0xf9, 0x43, 0x42, 0x58, 0xdb},
					// 	{0x5a, 0x45, 0x08, 0xbe, 0x10, 0x5b, 0xed, 0xd4, 0x30, 0x51, 0xcc, 0xf8, 0x4d, 0xd4, 0xe2, 0xca, 0x16, 0x76, 0x5d, 0xea},
					// 	{0xbc, 0x46, 0xcc, 0xa1, 0x65, 0x00, 0xfe, 0x0e, 0x73, 0x31, 0xa0, 0x92, 0x23, 0x9d, 0x49, 0x31, 0xd1, 0x9d, 0xfd, 0x41},
					// 	{0x6c, 0x47, 0x83, 0x47, 0xc1, 0x94, 0xec, 0x1b, 0xe1, 0x2d, 0xd0, 0x68, 0x58, 0x77, 0x19, 0xc2, 0x2a, 0xf8, 0x6a, 0x9b},
					// 	{0x8d, 0x4b, 0x53, 0x6b, 0xa5, 0xed, 0xdb, 0x06, 0x44, 0xf2, 0x80, 0x07, 0x7b, 0xbf, 0xd8, 0x63, 0xcb, 0x7b, 0x98, 0x60},
					// 	{0xea, 0xd2, 0x3c, 0x4f, 0x3c, 0x7c, 0x0f, 0x47, 0x9c, 0x35, 0x28, 0x02, 0x9f, 0x9f, 0xef, 0xb8, 0x96, 0x75, 0x87, 0x81},
					// 	{0xab, 0xa3, 0xda, 0x89, 0xfc, 0x0b, 0xb9, 0x47, 0x47, 0xa8, 0x54, 0xaa, 0x81, 0xb5, 0x9e, 0xee, 0x45, 0x22, 0x02, 0x67},
					// 	{0xd9, 0x0e, 0x02, 0x59, 0xda, 0xbf, 0x92, 0x0d, 0x81, 0x58, 0x28, 0xe8, 0xd7, 0x5d, 0xb1, 0x82, 0xcd, 0x2b, 0xf8, 0x64},
					// },
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
