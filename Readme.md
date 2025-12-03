# miniredis

**miniredis** is a minimal Redis-like server implemented in Go for educational purposes.

<img width="879" height="459" alt="image" src="https://github.com/user-attachments/assets/dceb7a2e-8b47-46e0-9e7c-8edcb8bf5c2b" />

It implements a subset of the Redis protocol and supports the following basic commands:

- `SET key value`: Set a value to a key.
- `GET key`: Get the value for a key.
- `CLIENT`: (Partial/Stub implementation)
- `HELLO`: (Partial/Stub implementation)

### Features

- RESP protocol parsing via [tidwall/resp](https://github.com/tidwall/resp)
- Compatible with the [go-redis](https://github.com/redis/go-redis) client (basic usage)
- Supports concurrent connections

### Usage

#### Build & Run

```sh
go build -o miniredis
./miniredis
```

By default, it listens on `:5001`. You can specify a custom address:

```sh
./miniredis -listenAddr=":6380"
```

#### Example: Using with go-redis

You can use miniredis like a real Redis server. Here’s an example test (see `server_test.go`):

```go
import (
    "context"
    "github.com/redis/go-redis/v9"
)

rdb := redis.NewClient(&redis.Options{
    Addr: "localhost:5001",
    Password: "",
    DB: 0,
})

err := rdb.Set(context.Background(), "foo", "bar", 0).Err()
val, err := rdb.Get(context.Background(), "foo").Result()
fmt.Println(val) // Output: bar
```

#### Example with Redis CLI

You can also use the official `redis-cli`:

```sh
redis-cli -p 5001
> set mykey myval
OK
> get mykey
"myval"
```

### Project Structure

- `main.go`: entrypoint, server startup and main loop
- `peer.go`: connection handler, command parsing from client
- `proto.go`: RESP parsing helpers, command definitions
- `server.go`: main server state, command processing loop

### Limitations

- Only SET, GET, CLIENT, and HELLO commands are implemented.
- No persistence, expiration, authentication, or advanced data types.
- `CLIENT` and `HELLO` are minimally supported for compatibility.
- Not production ready.

### License

MIT

### Credits

- [tidwall/resp](https://github.com/tidwall/resp) for RESP protocol parsing

---

