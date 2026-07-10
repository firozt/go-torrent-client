package shared

import (
	"encoding/hex"
	"fmt"
)

// example hexstr infohash
// 2D0149BC70F751A8680AAEDBBB3B43F84814FE73
func HexstrToHex(hexstr string) (*[]byte, error) {
	b, err := hex.DecodeString(hexstr)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// converts hexstr to infohash (20 byte)
// panics if conversion fails or length of conversion is not 20
func HexstrToInfohash(hexstr string) ([20]byte) {
	var res [20]byte

	conv, err := HexstrToHex(hexstr)

	if err != nil {
		panic(fmt.Sprintf("failed to convert string to hexstr value %s", hexstr))
	}

	if len(*conv) != 20 {
		panic(fmt.Sprintf("value '%s'not a valid infohash (too long)", hexstr))
	}

	copy(res[:], *conv)
	return res
}

