package torrentvalidator

import (
	"fmt"
	torrent "github.com/firozt/go-torrent/src/internal/Torrent"
)

// entry, takes bencode data and verifies all fields,
// makes sure its correct for either SFM or MFM
// returns a Torrent interface struct and error value
func ValidateBencodeData(data *torrent.RawTorrentData) (*torrent.TorrentFile, error) {
	// check its base

	torrentfile := &torrent.TorrentFile{}
	err := attemptParseBase(data, torrentfile)
	if err != nil {
		return nil, err
	}

	// check if it can be SFM
	parseErr := attemptParseSFM(data, torrentfile)
	if parseErr == nil {
		return torrentfile, err
	}

	// check if it can be MFM
	parseErr = attemptParseMFM(data, torrentfile)
	if parseErr == nil {
		return torrentfile, err
	}

	return nil, fmt.Errorf("data could not be parsed into either struct")
}

// checks wether it has the fields shared between SFM and MFM (base)
// MUST HAVE:
// announce
// info
// ---- piece length
// ---- piece
func attemptParseBase(data *torrent.RawTorrentData, torrentfile *torrent.TorrentFile) error {
	if data.Announce == "" {
		return fmt.Errorf("data could not be parsed into a base torrent file, announce is empty")
	}

	if !isInfoExist(data.Info) {
		return fmt.Errorf("data could not be parsed into a base torrent file, info is invalid or empty")
	}

	if data.Info.PieceLength < 0 {
		return fmt.Errorf("Piece length is negative, invalid for a torrentfile")
	}

	if data.CreationDate < 0 {
		return fmt.Errorf("Creation date is negative, invalid for a torrentfile")
	}

	validPieceVal, err := pieceStringToHashList(data.Info.Piece)

	if err != nil {
		return err
	}

	flattendList := flattenAnnounceList(data.AnnounceList)
	combinedAnnounce := append([]string{data.Announce}, flattendList...)
	torrentfile.Name = data.Info.Name
	torrentfile.Announce = combinedAnnounce
	torrentfile.PieceLength = uint64(data.Info.PieceLength)
	torrentfile.Pieces = validPieceVal
	torrentfile.InfoHash = data.InfoHash
	torrentfile.CreationDate = uint64(data.CreationDate)
	torrentfile.Length = uint64(data.Info.Length)

	return nil
}

func pieceStringToHashList(pieces string) ([][20]byte, error) {
	pieceBytes := []byte(pieces)

	if len(pieceBytes)%20 != 0 {
		return nil, fmt.Errorf("piece string is not a multiple of 20")
	}

	numHashes := len(pieces) / 20
	res := make([][20]byte, numHashes)
	for i := 0; i < numHashes; i++ {
		copy(res[i][:], pieceBytes[i*20:(i+1)*20])
	}

	return res, nil
}

func flattenAnnounceList(input [][]any) []string {
	var out []string
	for _, inner := range input {
		for _, item := range inner {
			// type assert each item to string
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
	}
	return out
}

func isInfoExist(info torrent.RawTorrentInfo) bool {
	if len(info.Piece) == 0 { // must have a piece string
		return false
	}
	if info.PieceLength == 0 { // must have piece length
		return false
	}

	return true
}

func attemptParseSFM(data *torrent.RawTorrentData, torrentfile *torrent.TorrentFile) error {
	// check SFM specific values are set
	if data.Info.Name == "" || data.Info.Length <= 0 {
		return fmt.Errorf("data could not be parsed into a SFM, info name is empty or info length is zero")
	}

	if torrentfile == nil {
		return fmt.Errorf("torrentfile is nil")
	}

	torrentfile.Name = data.Info.Name

	return nil
}

func attemptParseMFM(data *torrent.RawTorrentData, torrentfile *torrent.TorrentFile) error {
	// check for MFM specific values are set
	if len(data.Info.Files) < 1 { // checks that there exists atleast one file
		return fmt.Errorf("data could not be parsed into MFM, info files is empty")
	}
	files := []torrent.TorrentFileField{}

	for _, file := range data.Info.Files {
		files = append(files, file)
	}

	if torrentfile == nil {
		return fmt.Errorf("torrentfile is nil")
	}

	torrentfile.Files = files

	return nil
}
