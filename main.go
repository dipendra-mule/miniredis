package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"
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
		SyncRDB(conf)
		InitRDBTracker(state)
	}

	l, err := net.Listen("tcp", ":6379")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	log.Println("listening on :6379")

	var wg sync.WaitGroup
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("connection accepeted: ", conn.RemoteAddr())

		wg.Add(1)
		go func() {
			handleConn(conn, state)
			wg.Done()
		}()
	}
	wg.Wait()
}

func handleConn(conn net.Conn, state *AppState) {
	log.Println("accepeted new connection: ", conn.LocalAddr().String())
	for {
		r := Resp{sign: Array}
		if err := r.parseRespArr(conn); err != nil {
			log.Println(err)
			break
		}
		handle(conn, &r, state)
	}
	log.Println("connection closed: ", conn.LocalAddr().String())
}

type AppState struct {
	conf          *Config
	aof           *Aof
	bgsaveRunning bool
	dbCopy        map[string]string
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
