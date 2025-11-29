package main

import (
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
	cfg Config
	ln  net.Listener
}

func NewServer(cfg Config) *Server {
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = defaultListenAddr
	}
	return &Server{
		cfg: cfg,
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.cfg.ListenAddr)
	if err != nil {
		return err
	}
	go s.acceptLoop()
	s.ln = ln
	return nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			slog.Error("accept error: %v", err)
			continue
		}
	}
}

func main() {
	s := NewServer(Config{})
	s.Start()
}
