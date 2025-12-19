package main

import (
	"sync"
	"time"
)

type Database struct {
	store map[string]*Key
	mu    sync.RWMutex
}

func NewDatabase() *Database {
	return &Database{
		store: map[string]*Key{},
		mu:    sync.RWMutex{},
	}
}

func (db *Database) Set(k, v string) {
	db.store[k] = &Key{V: v}
}

func (db *Database) Delete(k string) {
	delete(db.store, k)
}

var DB = NewDatabase()

type Key struct {
	V   string
	Exp time.Time
}

func (key *Key) approxMemUsage(name string) int {
	stringHeader := 16
	expHeader := 24
	mapEntrySize := 32

	return stringHeader + len(name) + stringHeader + len(key.V) + expHeader + mapEntrySize
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
