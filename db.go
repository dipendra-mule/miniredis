package main

import (
	"log"
	"sync"
	"time"
)

type Database struct {
	store map[string]*Key
	mu    sync.RWMutex
	mem   int
}

func NewDatabase() *Database {
	return &Database{
		store: map[string]*Key{},
		mu:    sync.RWMutex{},
	}
}

func 

func (db *Database) Set(k, v string) {
	if old, ok := db.store[k]; ok {
		oldmem := old.approxMemUsage(k)
		db.mem -= oldmem
	}

	key := &Key{V: v}
	kmem := key.approxMemUsage(k)

	db.store[k] = key
	db.mem += kmem
	log.Println("db.mem", db.mem)
}

func (db *Database) Delete(k string) {
	key, ok := db.store[k]
	if !ok {
		return // fail gracefully
	}
	kmem := key.approxMemUsage(k)

	delete(db.store, k)
	db.mem -= kmem
	log.Println("db.mem", db.mem)
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
