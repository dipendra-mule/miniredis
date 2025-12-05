package main

import (
	"bufio"
	"fmt"
	"io"
)

type Writer struct {
	writer io.Writer
}

func NewWrite(w io.Writer) *Writer {
	return &Writer{
		writer: bufio.NewWriter(w),
	}
}

func (w *Writer) Write(r *Resp) {
	var reply string
	switch r.sign {
	case SimpleString:
		reply = fmt.Sprintf("%s%s\r\n", r.sign, r.str)
	case BulkString:
		reply = fmt.Sprintf("%s%d\r\n%s\r\n", r.sign, len(r.bulk), r.bulk)
	case Error:
		reply = fmt.Sprintf("%s%s\r\n", r.sign, r.err)
	case Null:
		reply = "$-1\r\n"
	}

	w.writer.Write([]byte(reply))
	w.writer.(*bufio.Writer).Flush()
}
