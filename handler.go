package main

import (
	"log"
	"maps"
	"path/filepath"
)

type Handler func(*Client, *Resp, *AppState) *Resp

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
	"AUTH":    auth,
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
	"auth":    auth,
}
var SafeCMDs = []string{
	"AUTH",
	"auth",
	"COMMAND",
}

func handle(c *Client, r *Resp, state *AppState) {
	cmd := r.arr[0].bulk
	handler, ok := Handlers[cmd]
	w := NewWrite(c.conn)
	if !ok {
		w.Write(&Resp{
			sign: Error,
			err:  "ERR invalid command",
		})
		w.Flush()
		return
	}

	if state.conf.requirepass && !c.authenticated && !contains(SafeCMDs, cmd) {
		w.Write(&Resp{
			sign: Error,
			err:  "ERR operation not permitted",
		})
		w.Flush()
		return
	}

	reply := handler(c, r, state)
	w.Write(reply)
	w.Flush()
}

func command(c *Client, r *Resp, state *AppState) *Resp {
	return &Resp{
		sign: SimpleString,
		str:  "OK",
	}
}

func set(c *Client, r *Resp, state *AppState) *Resp {
	args := r.arr[1:]
	if len(args) != 2 {
		return &Resp{
			sign: Error,
			err:  "ERR invalid args for 'SET'",
		}
	}
	k := args[0].bulk
	v := args[1].bulk

	// --------- db locked ---------
	DB.mu.Lock()
	DB.Set(k, v)
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
	// --------- db unlocked ---------

	return &Resp{
		sign: SimpleString,
		str:  "OK",
	}
}

func get(c *Client, r *Resp, state *AppState) *Resp {
	args := r.arr[1:]
	if len(args) != 1 {
		return &Resp{
			sign: Error,
			err:  "ERR invalid args for 'GET'",
		}
	}

	// --------- db locked ---------
	DB.mu.RLock()
	val, ok := DB.store[args[0].bulk]
	DB.mu.RUnlock()
	// --------- db unlocked ---------

	if !ok {
		return &Resp{
			sign: Null,
		}
	}
	return &Resp{
		sign: BulkString,
		bulk: val.V,
	}
}

func del(c *Client, r *Resp, state *AppState) *Resp {
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

func exists(c *Client, r *Resp, state *AppState) *Resp {
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

func keys(c *Client, r *Resp, state *AppState) *Resp {
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

func save(c *Client, r *Resp, state *AppState) *Resp {
	SaveRDB(state)
	return &Resp{
		sign: SimpleString, str: "OK",
	}
}

func bgsave(c *Client, r *Resp, state *AppState) *Resp {
	if state.bgsaveRunning {
		return &Resp{
			sign: Error, err: "ERR background save is already running",
		}
	}

	cp := make(map[string]*Key, len(DB.store))
	DB.mu.RLock()
	maps.Copy(cp, DB.store)
	DB.mu.RUnlock()

	state.bgsaveRunning = true
	state.dbCopy = cp
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

func dbsize(c *Client, r *Resp, state *AppState) *Resp {
	DB.mu.RLock()
	size := len(DB.store)
	DB.mu.RUnlock()

	return &Resp{
		sign: Integer,
		num:  size,
	}
}

func flushdb(c *Client, r *Resp, state *AppState) *Resp {
	DB.mu.Lock()
	DB.store = map[string]*Key{}
	DB.mu.Unlock()

	return &Resp{
		sign: SimpleString,
		str:  "OK",
	}
}

func auth(c *Client, r *Resp, state *AppState) *Resp {
	args := r.arr[1:]
	if len(args) != 1 {
		return &Resp{
			sign: Error,
			err:  "ERR invalid args for 'AUTH'",
		}
	}

	if state.conf.requirepass {
		if args[0].bulk != state.conf.password {
			c.authenticated = false
			return &Resp{
				sign: Error,
				err:  "ERR invalid password",
			}
		}
	}
	c.authenticated = true
	return &Resp{
		sign: SimpleString,
		str:  "OK",
	}
}
