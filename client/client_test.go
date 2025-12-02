package client

import (
	"context"
	"fmt"
	"testing"
)

func TestNewClient(t *testing.T) {
	c, err := NewClient("127.0.0.1:5000")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		if err := c.Set(context.Background(), fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i)); err != nil {
			t.Fatal(err)
		}

		val, err := c.Get(context.Background(), fmt.Sprintf("key%d", i))
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(val)
	}
}
