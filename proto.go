package main

import (
	"bytes"
	"fmt"

	"io"
	"log"

	"github.com/tidwall/resp"
)

const (
	CommandSet = "SET"
	CommandGet = "GET"
)

type Command interface {
}

type SetCommand struct {
	key, val []byte
}

type GetCommand struct {
	key []byte
}

func parseCommand(raw string) (Command, error) {
	rd := resp.NewReader(bytes.NewBufferString(raw))

	for {
		v, _, err := rd.ReadValue()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if v.Type() == resp.Array {
			switch v.Array()[0].String() {
			case CommandGet:
				if len(v.Array()) != 2 {
					return nil, fmt.Errorf("GET needs 1 args")
				}
				return GetCommand{
					key: v.Array()[1].Bytes(),
				}, nil
			case CommandSet:
				if len(v.Array()) != 3 {
					return nil, fmt.Errorf("SET needs 2 args")
				}
				return SetCommand{
					key: v.Array()[1].Bytes(),
					val: v.Array()[2].Bytes(),
				}, nil
			default:
				return nil, fmt.Errorf("unknown command: %s", v.Array()[0].String())
			}
		}
		return nil, fmt.Errorf("command to be implemented... ")
	}
	return nil, fmt.Errorf("no command found")
}
