package main

import (
	"net"
)

type Peer struct {
	conn  net.Conn
	msgCh chan []byte
}

func NewPeer(conn net.Conn, msgCh chan []byte) *Peer {
	return &Peer{
		conn:  conn,
		msgCh: msgCh,
	}
}

func (p *Peer) reedLoop() error {
	buf := make([]byte, 1024)
	// reader := bufio.NewReader(p.conn)
	for {
		n, err := p.conn.Read(buf)
		if err != nil {
			return err
		}
		// fmt.Println(string(buf[:n]))
		// fmt.Println(len(buf[:n]))
		msgBuff := make([]byte, n)
		copy(msgBuff, buf[:n])
		p.msgCh <- msgBuff
	}
}
