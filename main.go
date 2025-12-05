package main

import (
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	readConf("./redis.conf")

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
		handle(conn, &r)
	}
}
