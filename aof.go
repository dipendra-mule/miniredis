package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path"
)

type Aof struct {
	w    *Writer
	f    *os.File
	conf *Config
}

func NewAof(conf *Config) *Aof {
	aof := Aof{conf: conf}

	fp := path.Join(aof.conf.dir, aof.conf.aofFn)
	f, err := os.OpenFile(fp, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644) // owner (read-write) others (readonly)
	if err != nil {
		fmt.Println("cannot open:", fp)
		return &aof
	}
	aof.w = NewWrite(f)
	aof.f = f

	return &aof
}

func (aof *Aof) Sync() {
	for {
		r := Resp{}
		err := r.parseRespArr(aof.f)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println("unexpected err while reading AOF records")
			break
		}

		blankState := NewAppState(&Config{})
		c := Client{}
		set(&c, &r, blankState)
	}
}

func (aof *Aof) Rewrite(cp map[string]*Key) {
	// Re-route future AOF records to buffer
	var b bytes.Buffer
	aof.w = NewWrite(&b)

	// Clear file contents
	if err := aof.f.Truncate(0); err != nil {
		log.Println("aof rewrite - truncate error:", err)
		return
	}

	if _, err := aof.f.Seek(0, 0); err != nil {
		log.Println("aof rewrite - seek error:", err)
		return
	}

	// Rewrite all SET commands to file
	fwriter := NewWrite(aof.f)
	for k, v := range cp {
		cmd := Resp{sign: BulkString, bulk: "SET"}
		key := Resp{sign: BulkString, bulk: k}
		val := Resp{sign: BulkString, bulk: v.V}

		arr := Resp{sign: Array, arr: []Resp{
			cmd, key, val,
		}}
		fwriter.Write(&arr)
	}
	fwriter.Flush()

	// Re-route future AOF records back to file
	aof.w = NewWrite(aof.f)
}
