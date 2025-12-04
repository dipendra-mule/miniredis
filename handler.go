package main

import (
	"bufio"
	"fmt"
	"io"
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
	fmt.Println("reply", reply)
	w := NewWriter(conn)
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

	DB[args[0].bulk] = args[1].bulk
	fmt.Println("set req for", args[0].bulk, args[1].bulk)

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

	name := args[0].bulk
	val, ok := DB[name]
	if !ok {
		return &Resp{
			sign: Null,
		}
	}
	fmt.Println("get req for", args[0].bulk)
	return &Resp{
		sign: BulkString,
		bulk: val,
	}
}

type Writer struct {
	writer io.Writer
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		writer: bufio.NewWriter(w),
	}
}

func (w *Writer) Write(r *Resp) {
	var reply string
	switch r.sign {
	case SimpleString:
		reply = fmt.Sprintf("%s%s\r\n", r.sign, r.str)
		fmt.Println("simple string")
	case BulkString:
		reply = fmt.Sprintf("%s%d\r\n%s\r\n", r.sign, len(r.bulk), r.bulk)
		fmt.Println("bulk string")
	case Error:
		reply = fmt.Sprintf("%s%s\r\n", r.sign, r.err)
		fmt.Println("error string")
	case Null:
		reply = "$-1\r\n"
	}

	w.writer.Write([]byte(reply))
	fmt.Println("--------", reply)
	w.writer.(*bufio.Writer).Flush()
}
