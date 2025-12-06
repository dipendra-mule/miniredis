package main

import (
	"fmt"
	"io"
	"strconv"
)

type Sign string // todo convert to byte later

const (
	SimpleString Sign = "+"
	Error        Sign = "-"
	Integer      Sign = ":"
	BulkString   Sign = "$"
	Array        Sign = "*"
	Null         Sign = ""
)

type Resp struct {
	sign Sign
	num  int
	bulk string
	str  string
	arr  []Resp
	err  string
	// null    bool
}

// *3\r\n$3\r\nSET\r\n$3\r\nKey\r\n$5\r\nValue\r\n

func (r *Resp) parseRespArr(reader io.Reader) error {
	buf := make([]byte, 4)
	_, err := reader.Read(buf)
	if err != nil {
		return err
	}

	arrLen, err := strconv.Atoi(string(buf[1])) // 3
	if err != nil {
		return err
	}

	for range arrLen {
		bulk := r.parseBulkStr(reader)
		r.arr = append(r.arr, bulk)
	}
	return nil
}

func (r *Resp) parseBulkStr(reader io.Reader) Resp {
	buf := make([]byte, 4)
	reader.Read(buf)

	n, err := strconv.Atoi(string(buf[1]))
	if err != nil {
		fmt.Println(err)
		return Resp{}
	}

	bulkBuf := make([]byte, n+2)
	reader.Read(bulkBuf)

	bulk := string(bulkBuf[:n])
	return Resp{
		sign: BulkString,
		bulk: bulk,
	}
}
