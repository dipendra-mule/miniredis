package client

import (
	"context"
	"fmt"
	"log"
	"sync"
	"testing"
)

func TestNewClient(t *testing.T) {
	nClients := 10
	wg := sync.WaitGroup{}
	wg.Add(nClients)
	for i := 0; i < nClients; i++ {
		go func(i int) {
			c, err := NewClient("127.0.0.1:5001")
			if err != nil {
				log.Fatal(err)
			}
			defer c.Close()

			key := fmt.Sprintf("key%d", i)
			value := fmt.Sprintf("value%d", i)
			if err := c.Set(context.Background(), key, value); err != nil {
				log.Fatal(err)
			}
			val, err := c.Get(context.Background(), key)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("key: %s, val: %s\n", key, val)
			wg.Done()
		}(i)
	}
}

func TestClient2(t *testing.T) {
	c, err := NewClient("127.0.0.1:5001")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	key := "key"
	value := "value"
	if err := c.Set(context.Background(), key, value); err != nil {
		log.Fatal(err)
	}
	val, err := c.Get(context.Background(), key)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("key: %s, val: %s\n", key, val)
}
