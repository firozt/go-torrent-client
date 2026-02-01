package main

import (
	"fmt"
	"strings"

	bencodeparser "github.com/firozt/go-torrent/src/internal/BencodeParser"
)

func main() {
	fmt.Println("Starting")
	r := strings.NewReader("d8:announce41:http://bttracker.debian.org:6969/announce7:comment35:DebianCDfromcdimage.debian.org13:creationdatei1573903810ee")
	bencodeparser.Read(r)
}
