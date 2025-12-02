package main

import (
	"fmt"
	"log"
	"log/slog"
	"net"
)

var (
	defaultListenAddr = ":5000"
)

type Config struct {
	ListenAddr string
}

type Message struct {
	cmd  Command
	peer *Peer
}

type Server struct {
	cfg       Config
	peers     map[*Peer]bool
	ln        net.Listener
	addPeerCh chan *Peer
	quiteCh   chan struct{}
	msgCh     chan Message

	kv *KV
}

func NewServer(cfg Config) *Server {
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = defaultListenAddr
	}
	return &Server{
		cfg:       cfg,
		peers:     make(map[*Peer]bool),
		addPeerCh: make(chan *Peer),
		quiteCh:   make(chan struct{}),
		msgCh:     make(chan Message),
		kv:        NewKV(),
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.cfg.ListenAddr)
	if err != nil {
		return err
	}
	s.ln = ln
	go s.loop()

	slog.Info("server started", "listen_addr", s.cfg.ListenAddr)

	return s.acceptLoop()
}

func (s *Server) handleMsg(msg Message) error {

	switch v := msg.cmd.(type) {
	case SetCommand:
		return s.kv.Set(v.key, v.val)
	case GetCommand:
		val, ok := s.kv.Get(v.key)
		if !ok {
			return fmt.Errorf("key not found")
		}
		_, err := msg.peer.Send(val) // write directly to peer
		if err != nil {
			slog.Error("peer send error: %v", "err", err)
		}
	}
	return nil
}

func (s *Server) loop() {
	for {
		select {
		case rawMsg := <-s.msgCh:
			if err := s.handleMsg(rawMsg); err != nil {
				slog.Error("handle raw msg error: %v", "err", err)
				continue
			}
		case <-s.quiteCh:
			return
		case peer := <-s.addPeerCh:
			s.peers[peer] = true
		}
	}
}

func (s *Server) acceptLoop() error {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			slog.Error("accept error: %v", "err", err)
			continue
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	peer := NewPeer(conn, s.msgCh)
	s.addPeerCh <- peer
	if err := peer.reedLoop(); err != nil {
		slog.Error("read error: %v", "err", err, "peer", conn.RemoteAddr())
		return
	}
}

func main() {
	s := NewServer(Config{})
	log.Fatal(s.Start())
}
