package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

func main() {
	log.Println("reading config file")
	conf := readConf("./redis.conf")
	state := NewAppState(conf)

	if conf.aofEnabled {
		log.Println("syncing AOF records")
		state.aof.Sync()
	}

	if len(conf.rdb) > 0 {
		InitRDBTracker(conf)
	}

	l, err := net.Listen("tcp", ":6379")
	fmt.Println("server is started on port 6379")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	conn, err := l.Accept()
	fmt.Println("someone got connected", conn.RemoteAddr())

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer conn.Close()

	for {
		r := Resp{sign: Array}
		r.parseRespArr(conn)
		handle(conn, &r, state)
	}
}

type AppState struct {
	conf *Config
	aof  *Aof
}

func NewAppState(conf *Config) *AppState {
	state := AppState{
		conf: conf,
	}

	if conf.aofEnabled {
		state.aof = NewAof(conf)

		if conf.aofFSync == EverySec {
			go func() {
				t := time.NewTicker(time.Second)
				defer t.Stop()

				for range t.C {
					state.aof.w.Flush()
				}
			}()
		}
	}

	return &state
}
