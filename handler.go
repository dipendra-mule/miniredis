package main

import (
	"log"
	"maps"
	"path/filepath"
	"strconv"
	"time"
)

type Handler func(*Client, *Resp, *AppState) *Resp

var Handlers = map[string]Handler{
	"SET":        set,
	"GET":        get,
	"DEL":        del,
	"COMMAND":    command,
	"EXISTS":     exists,
	"KEYS":       keys,
	"SAVE":       save,
	"BGSAVE":     bgsave,
	"DBSIZE":     dbsize,
	"FLUSHDB":    flushdb,
	"AUTH":       auth,
	"EXPIRE":     expire,
	"TTL":        ttl,
	"BGWRITEAOF": bgwriteaof,
	"set":        set,
	"get":        get,
	"del":        del,
	"exists":     exists,
	"keys":       keys,
	"save":       save,
	"bgsave":     bgsave,
	"dbsize":     dbsize,
	"size":       dbsize,
	"flushdb":    flushdb,
	"auth":       auth,
	"expire":     expire,
	"ttl":        ttl,
	"bgwriteaof": bgwriteaof,
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
	if val.Exp.Unix() != UNIX_TS_EPOCH && time.Until(val.Exp).Seconds() <= 0 {
		DB.mu.Lock()
		DB.Delete(args[0].bulk)
		DB.mu.Unlock()
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
	defer DB.mu.Unlock()
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
	defer DB.mu.RUnlock()
	size := len(DB.store)

	return &Resp{
		sign: Integer,
		num:  size,
	}
}

func flushdb(c *Client, r *Resp, state *AppState) *Resp {
	DB.mu.Lock()
	defer DB.mu.Unlock()
	DB.store = map[string]*Key{}

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

func expire(c *Client, r *Resp, state *AppState) *Resp {
	args := r.arr[1:]
	if len(args) != 2 {
		return &Resp{
			sign: Error,
			err:  "ERR invalid args for 'EXPIRE'",
		}
	}
	k := args[0].bulk
	secs := args[1].bulk
	// secs := args[1].num

	expSecs, err := strconv.Atoi(secs)
	if err != nil {
		return &Resp{
			sign: Error,
			err:  "ERR invalid value for 'EXPIRE'",
		}
	}

	DB.mu.Lock()
	defer DB.mu.Unlock()

	key, ok := DB.store[k]
	if !ok {
		return &Resp{
			sign: Integer,
			num:  0,
		}
	}
	key.Exp = time.Now().Add(time.Duration(expSecs) * time.Second)

	return &Resp{
		sign: Integer,
		num:  1,
	}
}

func ttl(c *Client, r *Resp, state *AppState) *Resp {
	args := r.arr[1:]
	if len(args) != 1 {
		return &Resp{
			sign: Error,
			err:  "ERR invalid args for 'TTL'",
		}
	}

	k := args[0].bulk
	DB.mu.Lock()
	defer DB.mu.Unlock()
	key, ok := DB.store[k]
	if !ok {
		return &Resp{
			sign: Integer,
			num:  -2,
		}
	}
	exp := key.Exp

	if exp.Unix() == UNIX_TS_EPOCH {
		return &Resp{
			sign: Integer,
			num:  -1,
		}
	}
	expSecs := int(time.Until(exp).Seconds())
	if expSecs <= 0 {

		DB.Delete(k)

		return &Resp{
			sign: Integer,
			num:  -2,
		}
	}

	return &Resp{
		sign: Integer,
		num:  expSecs,
	}
}

func bgwriteaof(c *Client, r *Resp, state *AppState) *Resp {
	go func() {
		DB.mu.RLock()
		defer DB.mu.RUnlock()
		cp := make(map[string]*Key, len(DB.store))
		maps.Copy(cp, DB.store)

		state.aof.Rewrite(cp)
	}()
	return &Resp{
		sign: SimpleString,
		str:  "Background AOF rewrite started",
	}
}
