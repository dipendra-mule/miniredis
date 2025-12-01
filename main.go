package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"time"

	"github.com/dipendra-mule/miniredis/client"
)

var (
	defaultListenAddr = ":5000"
)

type Config struct {
	ListenAddr string
}

type Server struct {
	cfg       Config
	peers     map[*Peer]bool
	ln        net.Listener
	addPeerCh chan *Peer
	quiteCh   chan struct{}
	msgCh     chan []byte
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
		msgCh:     make(chan []byte),
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

func (s *Server) handleRawMsg(rawMsg []byte) error {
	cmd, err := parseCommand(string(rawMsg))
	if err != nil {
		return err
	}
	switch v := cmd.(type) {
	case SetCommand:
		slog.Info("set command", "cmd", v.key, "val", v.val)
	}
	return nil
}

func (s *Server) loop() {
	for {
		select {
		case rawMsg := <-s.msgCh:
			if err := s.handleRawMsg(rawMsg); err != nil {
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
	slog.Info("new peer connected", "peer", conn.RemoteAddr())
	if err := peer.reedLoop(); err != nil {
		slog.Error("read error: %v", "err", err, "peer", conn.RemoteAddr())
		return
	}
}

func main() {
	go func() {
		s := NewServer(Config{})
		log.Fatal(s.Start())
	}()

	time.Sleep(time.Second)
	c := client.NewClient("127.0.0.1:5000")
	if err := c.Set(context.Background(), "key1", "value1"); err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Second)

}
