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
func ValidateBencodeData(data *bencodeparser.Bencode) (Torrent, error) {
	// check its base
	err := checkBaseTorrent(data)
	if err != nil {
		return nil, err
	}
	// check if it can be SFM
	SFMData, err := attemptParseSFM(data)
	if err == nil {
		return SFMData, err
	}

	// check if it can be MFM
	MFMData, err := attemptParseMFM(data)

	if err == nil {
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
func checkBaseTorrent(data *bencodeparser.Bencode) error {
	if data.Announce == "" {
		return fmt.Errorf("data could not be parsed into a base torrent file, announce is empty")
	}

	if !isInfoExist(data.Info) {
		return fmt.Errorf("data could not be parsed into a base torrent file, info is invalid or empty")
	}

	return nil
}

func isInfoExist(info bencodeparser.BencodeInfo) bool {
	if len(info.Piece) > 0 {
		return false
	}
	if info.PieceLength == 0 {
		return false
	}

	return true
}

func attemptParseSFM(data *bencodeparser.Bencode) (*TorrentFileSFM, error) {
	// check SFM specific values are set
	if data.Info.Name == "" || data.Info.Length == 0 {
		return &TorrentFileSFM{}, fmt.Errorf("data could not be parsed into a SFM, info name is empty or info length is zero")
	}

	base := torrentFile{
		Name:         data.Info.Name,
		Announce:     data.Announce,
		AnnounceList: data.AnnounceList,
		PieceLength:  data.Info.PieceLength,
		Pieces:       data.Info.Piece,
		InfoHash:     data.InfoHash,
		CreationDate: data.CreationDate,
	}

	sfm := &TorrentFileSFM{
		torrentFile: base,
		Length:      data.Info.Length,
	}

	return sfm, nil
}

func attemptParseMFM(data *bencodeparser.Bencode) (TorrentFileMFM, error) {
	// check for MFM specific values are set
	if len(data.Info.Files) < 1 {
		return TorrentFileMFM{}, fmt.Errorf("data could not be parsed into MFM, info files is empty")
	}
	base := torrentFile{
		Name:         data.Info.Name,
		Announce:     data.Announce,
		AnnounceList: data.AnnounceList,
		InfoHash:     data.InfoHash,
		CreationDate: data.CreationDate,
		PieceLength:  data.Info.PieceLength,
		Pieces:       data.Info.Piece,
	}

	mfm := TorrentFileMFM{
		torrentFile: base,
		Files:       []bencodeparser.BencodeFile{},
	}

	for _, file := range data.Info.Files {
		mfm.Files = append(mfm.Files, file)
	}

	return mfm, nil

}
