package main

import (
	"fmt"
	"strings"

	bencodeparser "github.com/firozt/go-torrent/src/internal/BencodeParser"
)

func main() {
	fmt.Println("Starting")
	// data := strings.NewReader("d8:announce41:http://bttracker.debian.org:6969/announce7:comment35:\"DebianCDfromcdimage.debian.org\"13:creationdatei1573903810e4:infod6:lengthi351272960e4:name31:debian-10.2.0-amd64-netinst.iso12:piecelengthi262144eee")
	data := strings.NewReader("d8:announce41:http://bttracker.debian.org:6969/announce7:comment35:DebianCDfromcdimage.debian.org13:creationdatei1573903810ee")
	// data := strings.NewReader("d4:name5:Alice3:agei30ee")
	// data := strings.NewReader("d4:name5:Alice3:agei30e6:skillsl6:Python4:Goee")
	bencodeparser.Read(data)
}
