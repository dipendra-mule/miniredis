package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"

	"github.com/tidwall/resp"
)

const defaultListenAddr = ":5000"

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
	delPeerCh chan *Peer

	kv *KV
}

func NewServer(cfg Config) *Server {
	if len(cfg.ListenAddr) == 0 {
		cfg.ListenAddr = defaultListenAddr
	}
	return &Server{
		cfg:       cfg,
		peers:     make(map[*Peer]bool),
		addPeerCh: make(chan *Peer),
		quiteCh:   make(chan struct{}),
		msgCh:     make(chan Message),
		delPeerCh: make(chan *Peer),
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
	case ClientCommand:
		if err := resp.
			NewWriter(msg.peer.conn).
			WriteString("OK"); err != nil {
			return err
		}
	case SetCommand:
		if err := s.kv.Set(v.key, v.val); err != nil {
			return err
		}
		if err := resp.
			NewWriter(msg.peer.conn).
			WriteString("OK"); err != nil {
			return err
		}
	case GetCommand:
		val, ok := s.kv.Get(v.key)
		if !ok {
			return fmt.Errorf("key not found")
		}
		if err := resp.
			NewWriter(msg.peer.conn).
			WriteString(string(val)); err != nil {
			return err
		}
	case HelloCommand:
		spec := map[string]string{
			"server": "redis",
		}
		_, err := msg.peer.Send(respWriteMap(spec))
		if err != nil {
			return fmt.Errorf("peer send error: %s", err)
		}
	}
	return nil
}

func (s *Server) loop() {
	for {
		select {
		case msg := <-s.msgCh:
			if err := s.handleMsg(msg); err != nil {
				slog.Error("handle msg error: %v", "err", err)
			}
		case <-s.quiteCh:
			return
		case peer := <-s.addPeerCh:
			slog.Info("peer connected", "peer", peer.conn.RemoteAddr())
			s.peers[peer] = true
		case peer := <-s.delPeerCh:
			slog.Info("peer disconnected", "peer", peer.conn.RemoteAddr())
			delete(s.peers, peer)
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
	peer := NewPeer(conn, s.msgCh, s.delPeerCh)
	s.addPeerCh <- peer
	if err := peer.readLoop(); err != nil {
		slog.Error("read error: %v", "err", err, "peer", conn.RemoteAddr())
		return
	}
}

func main() {
	listenAdrr := flag.String("listenAddr", defaultListenAddr, "listen address of miniredis server")
	flag.Parse()
	s := NewServer(Config{
		ListenAddr: *listenAdrr,
	})
	log.Fatal(s.Start())
}
