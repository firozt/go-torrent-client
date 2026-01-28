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
			gotString, gotStart, _ := parseString([]byte(tc.input), 0)
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
			gotLength, gotStartOfString, gotError := getStringLength([]byte(tc.input), 0)

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
			got, gotEndIndex, err := parseInt([]byte(tc.input), 0)

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
				Info: BencodeInfo{
					Length:     163783,
					Name:       "alice.txt",
					PieceLenth: 16384,
					Piece:      [][20]byte{}, // skip comparison for this for now
				},
			},
			throwsError: false,
		},
	}

	for _, tc := range testcase {
		t.Run(tc.fileName, func(t *testing.T) {
			bencodeData, err := Read(readTestDataFile(tc.fileName))
			if !tc.throwsError && err != nil {
				t.Errorf("unexpected error thrown by Read - %s\n", err)
			}
			// set piece field to empty  as we skip this check for now
			bencodeData.InfoHash = [20]byte{}
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
