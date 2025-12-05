package main

import (
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
		set(&r, blankState)
	}
}
