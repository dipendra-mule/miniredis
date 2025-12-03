package main

import (
	"bytes"
	"fmt"

	"github.com/tidwall/resp"
)

const (
	CommandSet    = "set"
	CommandGet    = "get"
	CommandClient = "client"
	CommandHello  = "hello"
)

type Command interface {
}

type SetCommand struct {
	key, val []byte
}

type GetCommand struct {
	key []byte
}

type ClientCommand struct {
	value string
}

type HelloCommand struct {
	value string
}

func respWriteMap(m map[string]string) []byte {
	buf := &bytes.Buffer{}
	buf.WriteString("%" + fmt.Sprintf("%d\r\n", len(m)))
	rw := resp.NewWriter(buf)
	for k, v := range m {
		rw.WriteString(k)
		rw.WriteString(":" + v)
	}
	return buf.Bytes()
}

// func (p *Peer) parseCommad() (Command, error) {
// 	rd := resp.NewReader(p.conn)
// 	for {
// 		v, _, err := rd.ReadValue()
// 		if err == io.EOF {
// 			p.delCh <- p
// 			break
// 		}
// 		if err != nil {
// 			log.Fatal(err)
// 		}

// 		fmt.Printf("Received RESP value: %v\n", v)

// 		var cmd Command
// 		if v.Type() != resp.Array {
// 			arr := v.Array()
// 			c := arr[0].String()
// 			switch c {
// 			case CommandClient:
// 				cmd = ClientCommand{
// 					value: arr[1].String(),
// 				}

// 			case CommandHello:
// 				cmd = HelloCommand{
// 					value: arr[1].String(),
// 				}

// 			case CommandGet:
// 				if len(arr) != 2 {
// 					return nil, fmt.Errorf("GET needs 1 args")
// 				}
// 				cmd = GetCommand{
// 					key: arr[1].Bytes(),
// 				}

// 			case CommandSet:
// 				if len(arr) != 3 {
// 					return nil, fmt.Errorf("SET needs 2 args")
// 				}
// 				cmd = SetCommand{
// 					key: arr[1].Bytes(),
// 					val: arr[2].Bytes(),
// 				}

// 			default:
// 				return nil, fmt.Errorf("unknown command: %s", cmd)
// 			}
// 			return cmd, nil
// 		}
// 	}

// 	return nil, fmt.Errorf("no command found")
// }
