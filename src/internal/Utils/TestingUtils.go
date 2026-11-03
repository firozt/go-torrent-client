package utils

import (
	"encoding/json"
	"io"
	"os"
	"time"
)

type TrackerList struct {
	UDP         []string `json:"udp"`
	HTTP        []string `json:"http"`
	LastTimeRan time.Time      `json:"last"` //idk yet
}

func GetTorrentList() (*TrackerList, error) {
	const PATH_TO_LIST = "./../../../trackers.json"

	trackerListFile, err := os.Open(PATH_TO_LIST)

	if err != nil {
		return nil, err
	}

	defer trackerListFile.Close()

	info, _ := trackerListFile.Stat()
	data, err := io.ReadAll(trackerListFile)

	if err != nil {
		return nil, err
	}

	lastTimeRan := info.ModTime()

	var trackedList TrackerList
	json.Unmarshal(data, &trackedList)

	trackedList.LastTimeRan = lastTimeRan
	return &trackedList, err

}


