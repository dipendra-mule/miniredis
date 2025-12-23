package main

import (
	"errors"
	"sync"
	"time"
)

type Database struct {
	store map[string]*Key
	mu    sync.RWMutex
	mem   int64
}

func NewDatabase() *Database {
	return &Database{
		store: map[string]*Key{},
		mu:    sync.RWMutex{},
	}
}

func (db *Database) evictKeys(state *AppState, requiredMem int64) error {
	if state.conf.eviction == NoEvcition {
		return errors.New("maximum memory reached")
	}
	return nil
}

func (db *Database) Set(k, v string, state *AppState) {
	if old, ok := db.store[k]; ok {
		oldmem := old.approxMemUsage(k)
		db.mem -= oldmem
	}

	key := &Key{V: v}
	kmem := key.approxMemUsage(k)

	outOfMem := state.conf.maxmem > 0 && db.mem+kmem >= state.conf.maxmem
	if outOfMem {
		db.evictKeys(state, kmem)
	}

	db.store[k] = key
	db.mem += kmem

}

func (db *Database) Delete(k string) {
	key, ok := db.store[k]
	if !ok {
		return // fail gracefully
	}
	kmem := key.approxMemUsage(k)

	delete(db.store, k)
	db.mem -= kmem
}

var DB = NewDatabase()

type Key struct {
	V   string
	Exp time.Time
}

func (key *Key) approxMemUsage(name string) int64 {
	stringHeader := 16
	expHeader := 24
	mapEntrySize := 32

	return int64(stringHeader + len(name) + stringHeader + len(key.V) + expHeader + mapEntrySize)
}

type Transaction struct {
	cmds []*TxCommand
}

func NewTransaction() *Transaction {
	return &Transaction{}
}

type TxCommand struct {
	r       *Resp
	handler Handler
}
