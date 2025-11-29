package main

import (
	"fmt"
	"log"
	"log/slog"
	"net"
)

var (
	defaultListenAddr = ":6379"
)

type Config struct {
	ListenAddr string
}

type Server struct {
	cfg       Config
	peers     map[*Peer]bool
	ln        net.Listener
	addPeerCh chan *Peer
}

func NewServer(cfg Config) *Server {
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = defaultListenAddr
	}
	return &Server{
		cfg:       cfg,
		peers:     make(map[*Peer]bool),
		addPeerCh: make(chan *Peer),
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.cfg.ListenAddr)
	if err != nil {
		return err
	}
	s.ln = ln
	go s.loop()
	return s.acceptLoop()
}

func (s *Server) loop() {
	for {
		select {
		case peer := <-s.addPeerCh:
			s.peers[peer] = true
		default:
			fmt.Println("default")
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
	peer := NewPeer(conn)
	s.addPeerCh <- peer
	go peer.reedLoop()
}

func main() {
	s := NewServer(Config{})
	log.Fatal(s.Start())
}
