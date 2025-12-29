package main

import "net"

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
