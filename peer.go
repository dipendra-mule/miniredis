package main

import (
	"net"
)

type Peer struct {
	conn  net.Conn
	msgCh chan Message
}

func (p *Peer) Send(msg []byte) (int, error) {
	return p.conn.Write(msg)
}

func NewPeer(conn net.Conn, msgCh chan Message) *Peer {
	return &Peer{
		conn:  conn,
		msgCh: msgCh,
	}
}

func (p *Peer) reedLoop() error {
	for {
		cmd, err := p.parseCommad()
		if err != nil {
			return err
		}
		p.msgCh <- Message{
			cmd:  cmd,
			peer: p,
		}
	}
}
