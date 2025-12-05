package main

import "sync"

type KV struct {
	mu    sync.RWMutex
	store map[string]string
}

func NewKV() *KV {
	return &KV{
		store: map[string]string{},
		mu:    sync.RWMutex{},
	}
}

func (kv *KV) Set(key, val string) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	kv.store[key] = val
	return nil
}

func (kv *KV) Get(key string) (string, bool) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	val, ok := kv.store[key]
	return val, ok
}

var DB = NewKV()
