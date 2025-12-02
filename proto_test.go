package main

import (
	"fmt"
	"testing"
)

func TestParseCommand(t *testing.T) {
	// s := "*3\r\n$3\r\nSET\r\n$5\r\nkey1\r\n$5\r\nvalue1\r\n"
	// raw := "*3\r\n$3\r\nSET\r\n$5\r\nmykey\r\n$7\r\nmyvalue\r\n"
	raw := "*2\r\n$3\r\nGET\r\n$5\r\nmykey\r\n"

	cmd, err := parseCommand(raw)

	if err != nil {
		t.Fatalf("parseCommand error: %v", err)
	}
	fmt.Println(cmd)
	fmt.Println(string(cmd.(GetCommand).key))
}
