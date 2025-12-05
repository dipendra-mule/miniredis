package main

import (
	"encoding/gob"
	"log"
	"os"
	"path"
	"time"
)

type SnapshotTracker struct {
	keys   int
	ticker time.Ticker
	rdb    *RDBSnapshot
}

func NewSnapshotTracker(rdb *RDBSnapshot) *SnapshotTracker {
	return &SnapshotTracker{
		keys:   0,
		ticker: *time.NewTicker(time.Second * time.Duration(rdb.Secs)),
		rdb:    rdb,
	}
}

var trackers = []*SnapshotTracker{}

func InitRDBTracker(conf *Config) {
	for _, rdb := range conf.rdb {
		tracker := NewSnapshotTracker(&rdb)
		trackers = append(trackers, tracker)

		go func() {
			defer tracker.ticker.Stop()

			for range tracker.ticker.C {
				log.Printf("keys changed: %d - keys req to change: %d", tracker.keys, tracker.rdb.Secs)
				if tracker.keys >= tracker.rdb.KeysChanged {
					SaveRDB(conf)
				}
				tracker.keys = 0
			}
		}()
	}
}

func IncrRDBTracker() {
	for _, t := range trackers {
		t.keys++
	}
}

func SaveRDB(conf *Config) {
	fp := path.Join(conf.dir, conf.rdbFn)
	f, err := os.OpenFile(fp, os.O_CREATE|os.O_WRONLY, 0644) // owner (read-write), everyone (read)
	if err != nil {
		log.Println("error opening rdb file: ", err)
	}
	defer f.Close()

	err = gob.NewEncoder(f).Encode(&DB.store)
	if err != nil {
		log.Println("error saving rdb file: ", err)
		return
	}
	log.Println("saved RDB file")
}

func SyncRDB(conf *Config) {
	fp := path.Join(conf.dir, conf.rdbFn)
	f, err := os.OpenFile(fp, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		log.Println("error opening rdb file: ", err)
		return
	}
	defer f.Close()

	err = gob.NewDecoder(f).Decode(&DB.store)
	if err != nil {
		log.Println("error decoding rdb file: ", err)
		return
	}

}
