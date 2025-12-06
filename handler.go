package main

import (
	"fmt"
	"log"
	"net"
)

type Handler func(*Resp, *AppState) *Resp

var Handlers = map[string]Handler{
	"SET":     set,
	"GET":     get,
	"DEL":     del,
	"COMMAND": command,
	"set":     set,
	"get":     get,
	"del":     del,
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
		delete(DB.store, arg.bulk)
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
