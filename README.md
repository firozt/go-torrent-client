# Go Torrent Client
A bit-torrent client able to parse, download and validate files given a .torrent file, all controlable from the terminal using the TUI framework Bubbletea. This client follows all specifications listed by the [BitTorrent Spec](https://wiki.theory.org/BitTorrentSpecification)

## Contents

- [Project Goals](#project-goals)
- [Packages](#packages)
  - [BencodeParser](#bencodeparser-srcinternalbencodeparser)


## Project Goals
The main goals for this project are mostly personal, I want to learn how P2P networking works within Go and as a whole, furthemore how
parsing is designed and implemented.
Another area I wanted to delve into for this project is the Bubbletea TUI framework. Furthermore I want to improve my general Go programming
knowledge being my second project working with the programming language


## Packages
### BencodeParser `/src/internal/BencodeParser`  
Contains logic for mapping a .torrent file to a BencodeTorrent struct
Uses a recursive descent algorithm to parse each token and assign them to a key and value
The parser first forms an intermediate representation of the data in the form `map[string]any` as Bencode data can carry
any number of potential fields, all non essential fields will be ignore (essential fields being ones defined in the BencodeTorrent struct)
To create the general structure for this package I created a Context Free Grammar to represent the parsing shown below  
```ANTLR
value   -> integer | string | list | dict 
integer -> "i" int "e"
string  -> length ":" bytes
list    -> "l" value* "e"
dict    -> "d" (string value)* "e"
```
The CFG ruleset above is within the set of LL(1) grammars allowing it easily to be translated to code, and some applications such as ANTLR for Java
even does this very thing. However none exist for Go so I wrote everything by hand. The package also deals with large amount of bytes efficiently by
parsing buffer by buffer, whilst maintaining state between buffer changes. This is important as as the file grows the number of pieces (field in bencode data
will also grow)

Many state variables are maintaied throughout parsing defined by the BencodeParser struct below
```go

type BencodeParser struct {
	numDictsInInfoParsed int8 // number of dict value's parsed within the info key, used to understand when we are not in info anymore
	captureBytes         bool // tells the parser when to capture bytes for info_hash calculation
	infoBytes            []byte // holds all the bytes of the info dict
	buf                  []byte // buffer data
	buf_len              uint64 // number of actual data within buffer
	cur_idx              uint64 // current reading index within buffer
	reader               *io.Reader // reader of datasource (passed with Read call)
}
```
