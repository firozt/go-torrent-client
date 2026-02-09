package torrentvalidator

import (
	"fmt"

	bencodeparser "github.com/firozt/go-torrent/src/internal/BencodeParser"
)

type torrentFile struct {
	Name         string
	Announce     string
	AnnounceList []string
	InfoHash     [20]byte
	CreationDate uint64
	PieceLength  uint64
	Pieces       [][20]byte
}

type TorrentFileSFM struct {
	torrentFile
	Length uint64
}

type TorrentFileMFM struct {
	torrentFile
	Files []bencodeparser.BencodeFile
}

type Torrent interface {
	IsMultiFile() bool
}

func (torrentFile) isMultiFile() bool {
	panic("This struct should not be used to hold actual data, but should be abstract")
}
func (TorrentFileMFM) IsMultiFile() bool { return true }
func (TorrentFileSFM) IsMultiFile() bool { return false }

// entry, takes bencode data and verifies all fields,
// makes sure its correct for either SFM or MFM
// returns a Torrent interface struct and error value
func ValidateBencodeData(data *bencodeparser.BencodeTorrent) (Torrent, error) {
	// check its base
	base, err := attemptParseBase(data)
	if err != nil {
		return nil, err
	}

	// check if it can be SFM
	SFMData, err := attemptParseSFM(data)
	if err == nil {
		SFMData.torrentFile = *base
		return SFMData, err
	}

	// check if it can be MFM
	MFMData, err := attemptParseMFM(data)

	if err == nil {
		MFMData.torrentFile = *base
		return MFMData, err
	}

	return nil, fmt.Errorf("data could not be parsed into either struct")
}

// checks wether it has the fields shared between SFM and MFM (base)
// MUST HAVE:
// announce
// info
// ---- piece length
// ---- piece
func attemptParseBase(data *bencodeparser.BencodeTorrent) (*torrentFile, error) {
	if data.Announce == "" {
		return nil, fmt.Errorf("data could not be parsed into a base torrent file, announce is empty")
	}

	if !isInfoExist(data.Info) {
		return nil, fmt.Errorf("data could not be parsed into a base torrent file, info is invalid or empty")
	}

	if data.Info.PieceLength < 0 {
		return nil, fmt.Errorf("Piece length is negative, invalid for a torrentfile")
	}

	if data.CreationDate < 0 {
		return nil, fmt.Errorf("Creation date is negative, invalid for a torrentfile")
	}

	validPieceVal, err := pieceStringToHashList(data.Info.Piece)

	if err != nil {
		return nil, err
	}

	base := torrentFile{
		Name:         data.Info.Name,
		Announce:     data.Announce,
		AnnounceList: flattenAnnounceList(data.AnnounceList),
		PieceLength:  uint64(data.Info.PieceLength),
		Pieces:       validPieceVal,
		InfoHash:     data.InfoHash,
		CreationDate: uint64(data.CreationDate),
	}

	return &base, nil
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

func isInfoExist(info bencodeparser.BencodeInfo) bool {
	if len(info.Piece) == 0 { // must have a piece string
		return false
	}
	if info.PieceLength == 0 { // must have piece length
		return false
	}

	return true
}

func attemptParseSFM(data *bencodeparser.BencodeTorrent) (*TorrentFileSFM, error) {
	// check SFM specific values are set
	if data.Info.Name == "" || data.Info.Length <= 0 {
		return &TorrentFileSFM{}, fmt.Errorf("data could not be parsed into a SFM, info name is empty or info length is zero")
	}

	sfm := &TorrentFileSFM{
		Length: uint64(data.Info.Length),
	}

	return sfm, nil
}

func attemptParseMFM(data *bencodeparser.BencodeTorrent) (*TorrentFileMFM, error) {
	// check for MFM specific values are set
	if len(data.Info.Files) < 1 { // checks that there exists atleast one file
		return nil, fmt.Errorf("data could not be parsed into MFM, info files is empty")
	}
	mfm := TorrentFileMFM{
		Files: []bencodeparser.BencodeFile{},
	}

	for _, file := range data.Info.Files {
		mfm.Files = append(mfm.Files, file)
	}

	return &mfm, nil

}
