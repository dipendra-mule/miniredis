package client

import (
	"bytes"
	"context"
	"log"
	"net"

	"github.com/tidwall/resp"
)

type Client struct {
	addr string
	conn net.Conn
}

func NewClient(adrr string) (*Client, error) {
	conn, err := net.Dial("tcp", adrr)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return &Client{
		addr: adrr,
		conn: conn,
	}, nil
}

func (c *Client) Set(ctx context.Context, key, val string) error {
	buf := bytes.Buffer{}
	wr := resp.NewWriter(&buf)
	wr.WriteArray([]resp.Value{resp.StringValue("SET"), resp.StringValue(key), resp.StringValue(val)})
	_, err := c.conn.Write(buf.Bytes())
	return err
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	buf := bytes.Buffer{}
	wr := resp.NewWriter(&buf)
	wr.WriteArray([]resp.Value{resp.StringValue("GET"), resp.StringValue(key)})
	_, err := c.conn.Write(buf.Bytes())
	if err != nil {
		return " ", err
	}

	b := make([]byte, 1024)
	n, err := c.conn.Read(b)
	return string(b[:n]), err
}

func (c *Client) Close() error {
	defer c.conn.Close()
	return nil
}
