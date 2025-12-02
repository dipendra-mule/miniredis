package client

import (
	"bytes"
	"context"
	"io"
	"log"
	"net"

	"github.com/tidwall/resp"
)

type Client struct {
	addr string
	conn net.Conn
}

func NewClient(adrr string) *Client {
	conn, err := net.Dial("tcp", adrr)
	if err != nil {
		log.Fatal(err)
	}

	return &Client{
		addr: adrr,
		conn: conn,
	}
}

func (c *Client) Set(ctx context.Context, key, val string) error {
	if c.conn == nil {

	}

	buf := &bytes.Buffer{}
	wr := resp.NewWriter(buf)
	wr.WriteArray([]resp.Value{resp.StringValue("set"), resp.StringValue(key), resp.StringValue(val)})
	_, err := c.conn.Write(buf.Bytes())
	if err != nil {
		return err
	}
	io.Copy(c.conn, buf)
	return nil
}
