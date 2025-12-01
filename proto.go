package main

import (
	"bytes"
	"fmt"

	"io"
	"log"

	"github.com/tidwall/resp"
)

type Command interface {
}

type SetCommand struct {
	key, val []byte
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
			case "set":
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
