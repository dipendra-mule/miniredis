package main

import (
	"fmt"
	"log"
	"maps"
	"net"
	"path/filepath"
)

type Handler func(*Resp, *AppState) *Resp

var Handlers = map[string]Handler{
	"SET":     set,
	"GET":     get,
	"DEL":     del,
	"COMMAND": command,
	"EXISTS":  exists,
	"KEYS":    keys,
	"SAVE":    save,
	"BGSAVE":  bgsave,
	"DBSIZE":  dbsize,
	"FLUSHDB": flushdb,
	"set":     set,
	"get":     get,
	"del":     del,
	"exists":  exists,
	"keys":    keys,
	"save":    save,
	"bgsave":  bgsave,
	"dbsize":  dbsize,
	"size":    dbsize,
	"flushdb": flushdb,
}

func handle(conn net.Conn, r *Resp, state *AppState) {
	cmd := r.arr[0].bulk
	handler, ok := Handlers[cmd]
	if !ok {
		fmt.Println("invalid command :", cmd)
		return
	}

	reply := handler(r, state)
	w := NewWrite(conn)
	w.Write(reply)
	w.Flush()
}

func command(r *Resp, state *AppState) *Resp {
	return &Resp{
		sign: SimpleString,
		str:  "OK",
	}
}

func set(r *Resp, state *AppState) *Resp {
	args := r.arr[1:]
	if len(args) != 2 {
		return &Resp{
			sign: Error,
			err:  "ERR invalid args for 'SET'",
		}
	}

	k := args[0].bulk
	v := args[1].bulk
	DB.mu.Lock()
	DB.store[k] = v
	if state.conf.aofEnabled {
		log.Println("saving aof file")
		state.aof.w.Write(r)

		if state.conf.aofFSync == Always {
			state.aof.w.Flush()
		}
	}

	if len(state.conf.rdb) >= 0 {
		IncrRDBTracker()
	}
	DB.mu.Unlock()

	return &Resp{
		sign: SimpleString,
		str:  "OK",
	}
}

func get(r *Resp, state *AppState) *Resp {
	args := r.arr[1:]
	if len(args) != 1 {
		return &Resp{
			sign: Error,
			err:  "ERR invalid args for 'GET'",
		}
	}

	val, ok := DB.Get(args[0].bulk)

	if !ok {
		return &Resp{
			sign: Null,
		}
	}
	return &Resp{
		sign: BulkString,
		bulk: val,
	}
}

func del(r *Resp, state *AppState) *Resp {
	args := r.arr[1:]
	var n int

	DB.mu.Lock()
	for _, arg := range args {
		_, ok := DB.store[arg.bulk]
		if ok {
			delete(DB.store, arg.bulk)
			n++
		}
	}
	// if state.conf.aofEnabled {
	// 	log.Println("saving aof file")
	// 	state.aof.w.Write(r)

	// 	if state.conf.aofFSync == Always {
	// 		state.aof.w.Flush()
	// 	}
	// }
	DB.mu.Unlock()

	return &Resp{
		sign: Integer,
		num:  n,
	}
}

func exists(r *Resp, state *AppState) *Resp {
	args := r.arr[1:]
	var n int

	DB.mu.Lock()
	for _, arg := range args {
		_, ok := DB.store[arg.bulk]
		if ok {
			n++
		}
	}
	DB.mu.Unlock()
	return &Resp{
		sign: Integer,
		num:  n,
	}
}

func keys(r *Resp, state *AppState) *Resp {
	args := r.arr[1:]
	if len(args) > 1 {
		return &Resp{
			sign: Error,
			err:  "ERR invalid args for 'KEYS'",
		}
	}
	pattern := args[0].bulk

	DB.mu.RLock()
	var matches []string
	for key := range DB.store {
		matched, err := filepath.Match(pattern, key)
		if err != nil {
			log.Printf("error matching keys: (pattern: %s, key: %s)- %s", pattern, key, err)
			continue
		}
		if matched {
			matches = append(matches, key)
		}
	}
	DB.mu.RUnlock()

	reply := &Resp{
		sign: Array,
	}

	for _, m := range matches {
		reply.arr = append(reply.arr, Resp{sign: BulkString, bulk: m})
	}
	return reply
}

func save(r *Resp, state *AppState) *Resp {
	SaveRDB(state)
	return &Resp{
		sign: SimpleString, str: "OK",
	}
}

func bgsave(r *Resp, state *AppState) *Resp {
	if state.bgsaveRunning {
		return &Resp{
			sign: Error, err: "ERR background save is already running",
		}
	}

	c := make(map[string]string, len(DB.store))
	DB.mu.RLock()
	maps.Copy(c, DB.store)
	DB.mu.RUnlock()

	state.bgsaveRunning = true
	state.dbCopy = c
	go func() {
		defer func() {
			state.bgsaveRunning = false
			state.dbCopy = nil
		}()
		SaveRDB(state)
	}()

	return &Resp{
		sign: SimpleString,
		str:  "OK",
	}
}

func dbsize(r *Resp, state *AppState) *Resp {
	DB.mu.RLock()
	size := len(DB.store)
	DB.mu.RUnlock()

	return &Resp{
		sign: Integer,
		num:  size,
	}
}

func flushdb(r *Resp, state *AppState) *Resp {
	DB.mu.Lock()
	DB.store = map[string]string{}
	DB.mu.Unlock()

	return &Resp{
		sign: SimpleString,
		str:  "OK",
	}
}
