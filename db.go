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
