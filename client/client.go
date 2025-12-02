package client

import (
	"bytes"
	"context"
	"net"

	"github.com/tidwall/resp"
)

type Client struct {
	addr string
	conn net.Conn
}

func NewClient(adrr string) *Client {
	return &Client{
		addr: adrr,
	}
}

func (c *Client) Set(ctx context.Context, key, val string) error {
	conn, err := net.Dial("tcp", c.addr)
	if err != nil {
		return err
	}

	buf := bytes.Buffer{}
	wr := resp.NewWriter(&buf)
	wr.WriteArray([]resp.Value{resp.StringValue("SET"), resp.StringValue(key), resp.StringValue(val)})
	_, err = conn.Write(buf.Bytes())
	return err
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	conn, err := net.Dial("tcp", c.addr)
	if err != nil {
		return " ", err
	}

	buf := bytes.Buffer{}
	wr := resp.NewWriter(&buf)
	wr.WriteArray([]resp.Value{resp.StringValue("GET"), resp.StringValue(key)})
	_, err = conn.Write(buf.Bytes())
	if err != nil {
		return " ", err
	}

	b := make([]byte, 1024)
	n, err := conn.Read(b)
	return string(b[:n]), err
}
