package main

import (
	"fmt"
	"testing"
)

func TestParseCommand(t *testing.T) {
	// s := "*3\r\n$3\r\nSET\r\n$5\r\nkey1\r\n$5\r\nvalue1\r\n"
	raw := "*3\r\n$3\r\nset\r\n$6\r\nleader\r\n$7\r\nCharlie\r\n"

	cmd, err := parseCommand(raw)
	if err != nil {
		t.Fatalf("parseCommand error: %v", err)
	}
	fmt.Println(cmd)
}
