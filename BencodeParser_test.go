package main

import "testing"

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
			gotString, gotStart, _ := parseString([]byte(tc.input))
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
			gotLength, gotStartOfString, gotError := getStringLength([]byte(tc.input))

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
