# Go Torrent Client

## Summary
This project will try to cover all fundemntals of P2P networking as well as how torrenting works. My aims for this project is to create a fully working torrenting
client written primarily in the Go programming langugae without the use of any third party libraries.


## Packages
### BencodeParser
- `/src/internal/BencodeParser`  
Contains logic for mapping a .torrent file to a BencodeTorrent struct
Uses a recursive descent algorithm to parse each token and assign them to a key and value
The parser first forms an intermediate representation of the data in the form `map[string]anY` as bencode data can carry
any number of potential fields, all non essential fields will be ignore (essential fields being ones defined in the BencodeTorrent struct)
To create the general structure for this package I created a Context Free Grammar to represent the parsing shown below  
```
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

