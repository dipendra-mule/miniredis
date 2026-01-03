package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

var UNIX_TS_EPOCH int64 = -62135596800

func main() {
	log.Println("reading config file")
	conf := readConf("./redis.conf")

	// Wire config → RESP parser
	if conf.maxBulkSize > 0 {
		MaxBulkSize = conf.maxBulkSize
	}
	if conf.maxCommandSize > 0 {
		MaxCommandSize = conf.maxCommandSize
	}
	if conf.maxCommandArgs > 0 {
		MaxCommandArgs = conf.maxCommandArgs
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

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("connection accepeted: ", conn.RemoteAddr())

		go func() {
			handleConn(conn, state)

		}()
	}
}

func handleConn(conn net.Conn, state *AppState) {
	log.Println("accepeted new connection: ", conn.LocalAddr().String())
	c := NewClient(conn)
	rd := bufio.NewReader(conn)
	for {
		r := Resp{sign: Array}
		if err := r.parseRespArr(rd); err != nil {
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
