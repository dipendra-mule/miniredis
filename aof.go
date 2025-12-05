package main

import (
	"fmt"
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
