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

	buf := &bytes.Buffer{}
	wr := resp.NewWriter(buf)
	wr.WriteArray([]resp.Value{resp.StringValue("set"), resp.StringValue(key), resp.StringValue(val)})
	_, err = conn.Write(buf.Bytes())
	return err
}
