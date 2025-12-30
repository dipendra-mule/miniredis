package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

var UNIX_TS_EPOCH int64 = -62135596800

func main() {
	log.Println("reading config file")
	conf := readConf("./redis.conf")

	// Wire config → RESP parser
	if conf.MaxBulkSize > 0 {
		MaxBulkSize = conf.MaxBulkSize
	}
	if conf.MaxCommandSize > 0 {
		MaxCommandSize = conf.MaxCommandSize
	}
	if conf.MaxCommandArgs > 0 {
		MaxCommandArgs = conf.MaxCommandArgs
	}

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
}

func handleConn(conn net.Conn, state *AppState) {
	log.Println("accepeted new connection: ", conn.LocalAddr().String())
	c := NewClient(conn)

	for {
		r := Resp{sign: Array}

		if err := r.parseRespArr(conn); err != nil {
			// ✅ Send protocol error
			w := NewWrite(conn)
			w.Write(&Resp{
				sign: Error,
				err:  err.Error(),
			})
			w.Flush()

			log.Println(err)

			// 🔥 FORCE CLIENT WRITE TO FAIL (TCP RST)
			if tcp, ok := conn.(*net.TCPConn); ok {
				tcp.SetLinger(0)
			}

			conn.Close()
			break
		}

		handle(c, &r, state)
	}

	log.Println("connection closed: ", conn.LocalAddr().String())
}

type Client struct {
	conn          net.Conn
	authenticated bool
	tx            *Transaction
}

func NewClient(conn net.Conn) *Client {
	return &Client{
		conn: conn,
	}
}

type AppState struct {
	conf          *Config
	aof           *Aof
	bgsaveRunning bool
	dbCopy        map[string]*Key
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
