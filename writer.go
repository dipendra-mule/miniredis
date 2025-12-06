package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
)

type Writer struct {
	writer io.Writer
}

func NewWrite(w io.Writer) *Writer {
	return &Writer{
		writer: bufio.NewWriter(w),
	}
}

func (w *Writer) Deserialize(r *Resp) (reply string) {
	switch r.sign {
	case Array:
		reply = fmt.Sprintf("%s%d\r\n", r.sign, len(r.arr))
		for _, sub := range r.arr {
			reply += w.Deserialize(&sub)
		}
	case SimpleString:
		reply = fmt.Sprintf("%s%s\r\n", r.sign, r.str)
	case BulkString:
		reply = fmt.Sprintf("%s%d\r\n%s\r\n", r.sign, len(r.bulk), r.bulk)
	case Integer:
		reply = fmt.Sprintf("%s%d\r\n", r.sign, r.num)
	case Error:
		reply = fmt.Sprintf("%s%s\r\n", r.sign, r.err)
	case Null:
		reply = "$-1\r\n"
	default:
		log.Println("invalid typ received")
		return reply
	}
	return reply
}

func (w *Writer) Write(r *Resp) {
	reply := w.Deserialize(r)
	w.writer.Write([]byte(reply))
}

func (w *Writer) Flush() {
	w.writer.(*bufio.Writer).Flush()
}
