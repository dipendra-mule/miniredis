package main

import (
	"fmt"
	"net"
)

type Handler func(*Resp) *Resp

var Handlers = map[string]Handler{
	"SET":     set,
	"GET":     get,
	"COMMAND": command,
}

func handle(conn net.Conn, r *Resp) {
	cmd := r.arr[0].bulk
	handler, ok := Handlers[cmd]
	if !ok {
		fmt.Println("invalid command :", cmd)
		return
	}

	reply := handler(r)
	w := NewWrite(conn)
	w.Write(reply)
}
func command(r *Resp) *Resp {
	return &Resp{
		sign: SimpleString,
		str:  "OK",
	}
}
func set(r *Resp) *Resp {
	args := r.arr[1:]
	if len(args) != 2 {
		return &Resp{
			sign: Error,
			err:  "ERR invalid args for 'SET'",
		}
	}

	k := args[0].bulk
	v := args[1].bulk
	err := DB.Set(k, v)
	if err != nil {
		fmt.Println("failed to set kv to db", "err:", err)
	}

	return &Resp{
		sign: SimpleString,
		str:  "OK",
	}
}

func get(r *Resp) *Resp {
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
