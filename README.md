# MiniRedis

A lightweight, Redis-compatible in-memory key-value store written in Go. MiniRedis implements the RESP (Redis Serialization Protocol) protocol and provides core Redis functionality including persistence, transactions, and authentication.

## Features

### Core Commands
- **SET** - Set a key-value pair
- **GET** - Retrieve a value by key
- **DEL** - Delete one or more keys
- **EXISTS** - Check if one or more keys exist
- **KEYS** - List all keys matching a pattern
- **DBSIZE** - Get the number of keys in the database
- **FLUSHDB** - Remove all keys from the database

### Expiration
- **EXPIRE** - Set a timeout on a key (in seconds)
- **TTL** - Get the remaining time to live of a key

### Persistence
- **RDB Snapshots** - Point-in-time snapshots of the database
  - Automatic snapshots based on time and key change thresholds
  - Manual snapshots via `SAVE` command
  - Background snapshots via `BGSAVE` command
- **AOF (Append-Only File)** - Log of all write operations
  - Configurable fsync modes: `always`, `everysec`, `no`
  - Background AOF rewrite via `BGWRITEAOF` command

### Transactions
- **MULTI** - Start a transaction
- **EXEC** - Execute all commands in a transaction
- **DISCARD** - Cancel a transaction

### Authentication
- **AUTH** - Authenticate with password (if `requirepass` is set in config)

### Other
- **COMMAND** - Basic command support
- **BGWRITEAOF** - Trigger background AOF rewrite

## Architecture

### Components

#### `main.go`
- Entry point of the application
- TCP server listening on port 6379 (Redis default)
- Connection handling and client management
- Application state initialization
- AOF sync scheduling (for `everysec` mode)

#### `handler.go`
- Command routing and execution
- Implements all Redis command handlers
- Transaction management
- Authentication checks

#### `db.go`
- In-memory database structure with thread-safe operations
- Key-value storage with expiration support
- Memory tracking and eviction policies (currently only `noeviction` implemented)
- Transaction command queue

#### `resp.go`
- RESP protocol parser
- Parses incoming Redis protocol messages
- Handles arrays and bulk strings

#### `writer.go`
- RESP protocol serializer
- Converts responses to Redis protocol format
- Buffered writing for performance

#### `aof.go`
- AOF persistence implementation
- AOF file replay on startup
- Background AOF rewrite functionality

#### `rdb.go`
- RDB snapshot implementation
- Automatic snapshot scheduling based on configuration
- Snapshot tracking (keys changed counter)
- SHA256 checksum verification for data integrity

#### `conf.go`
- Configuration file parser (`redis.conf`)
- Supports Redis-compatible configuration format
- Memory size parsing (supports KB, MB, GB suffixes)

#### `utils.go`
- Utility functions (e.g., string contains check)

## Configuration

MiniRedis uses a `redis.conf` configuration file (similar to Redis). Create a `redis.conf` file in the project root with the following options:

### Configuration Options

```conf
# Data directory for persistence files
dir ./data

# AOF Configuration
appendonly yes                    # Enable AOF persistence
appendfilename backup.aof         # AOF filename
appendfsync always                # Fsync mode: always, everysec, or no

# RDB Configuration
save 900 1                        # Save if 1 key changed in 900 seconds
save 300 10                       # Save if 10 keys changed in 300 seconds
dbfilename backup.rdb             # RDB filename

# Authentication
requirepass yourpassword          # Set password (commented = disabled)

# Memory Management
maxmemory 256mb                   # Maximum memory (supports KB, MB, GB)
maxmemory-policy noeviction      # Eviction policy (currently only noeviction)
```

### Configuration Details

- **dir**: Directory where RDB and AOF files are stored
- **appendonly**: Enable/disable AOF persistence (`yes` or `no`)
- **appendfilename**: Name of the AOF file
- **appendfsync**: 
  - `always`: Fsync after every write (safest, slowest)
  - `everysec`: Fsync every second (balanced)
  - `no`: Let OS decide when to fsync (fastest, less safe)
- **save**: RDB snapshot trigger (`save <seconds> <keys_changed>`)
  - Multiple `save` directives can be specified
  - Snapshot is created if `keys_changed` keys are modified within `seconds`
- **dbfilename**: Name of the RDB snapshot file
- **requirepass**: Password for authentication (if set, all commands except AUTH require authentication)
- **maxmemory**: Maximum memory usage (supports `b`, `kb`, `mb`, `gb` suffixes)
- **maxmemory-policy**: Currently only `noeviction` is implemented

## Installation

### Prerequisites
- Go 1.22.2 or later

### Build

```bash
cd miniredis
go build -o miniredis
```

### Run

```bash
./miniredis
```

The server will start listening on `:6379` by default.

## Usage

### Using redis-cli

MiniRedis is compatible with the standard `redis-cli` tool:

```bash
# Connect to MiniRedis
redis-cli

# Set a key
SET mykey "Hello World"

# Get a key
GET mykey

# Set expiration
SET mykey "Hello" EX 60
EXPIRE mykey 60

# Check TTL
TTL mykey

# Transaction example
MULTI
SET key1 "value1"
SET key2 "value2"
EXEC

# Authentication (if password is set)
AUTH yourpassword
```

### Using Go Client

You can use any Redis client library that supports RESP protocol:

```go
import "github.com/go-redis/redis/v8"

client := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

err := client.Set(ctx, "key", "value", 0).Err()
val, err := client.Get(ctx, "key").Result()
```

## Persistence

### RDB Snapshots

RDB snapshots are created automatically based on the `save` configuration directives. The snapshot tracks the number of keys changed and creates a snapshot when thresholds are met.

- **Manual snapshot**: Use `SAVE` command (blocks until complete)
- **Background snapshot**: Use `BGSAVE` command (non-blocking)
- **Automatic snapshots**: Configured via `save` directives in `redis.conf`

RDB files use Go's `gob` encoding and include SHA256 checksums for integrity verification.

### AOF (Append-Only File)

AOF logs every write operation and replays them on startup to restore the database state.

- **AOF Rewrite**: Use `BGWRITEAOF` to compact the AOF file
- **Fsync modes**: Control durability vs performance trade-off
- **Startup recovery**: AOF is automatically replayed when the server starts

## Thread Safety

MiniRedis uses `sync.RWMutex` for thread-safe database operations:
- Read operations use `RLock()` for concurrent reads
- Write operations use `Lock()` for exclusive access
- Each client connection is handled in a separate goroutine

## Memory Management

MiniRedis tracks approximate memory usage for each key-value pair. When `maxmemory` is set:
- Memory usage is tracked on SET and DELETE operations
- Currently, only `noeviction` policy is implemented (returns error when memory limit is reached)
- Future eviction policies can be added (LRU, LFU, etc.)

## Limitations

- **Single database**: Only one database (no SELECT command)
- **String values only**: Currently supports only string values (no lists, sets, hashes, etc.)
- **Limited eviction**: Only `noeviction` policy is implemented
- **No replication**: No master-slave replication support
- **No clustering**: No cluster mode support
- **No pub/sub**: No publish/subscribe functionality

## Protocol

MiniRedis implements the RESP (Redis Serialization Protocol) specification:
- Simple Strings: `+OK\r\n`
- Errors: `-ERR message\r\n`
- Integers: `:123\r\n`
- Bulk Strings: `$5\r\nhello\r\n`
- Arrays: `*2\r\n$3\r\nSET\r\n$3\r\nkey\r\n`
- Null: `$-1\r\n`

## Development

### Project Structure

```
miniredis/
├── main.go          # Entry point, server setup
├── handler.go       # Command handlers
├── db.go            # Database implementation
├── resp.go          # RESP protocol parser
├── writer.go        # RESP protocol serializer
├── aof.go           # AOF persistence
├── rdb.go           # RDB snapshots
├── conf.go          # Configuration parser
├── utils.go         # Utility functions
├── redis.conf       # Configuration file
├── go.mod           # Go module definition
└── data/            # Persistence files directory
    ├── backup.rdb
    └── backup.aof
```

### Adding New Commands

1. Add handler function in `handler.go`:
```go
func mycommand(c *Client, r *Resp, state *AppState) *Resp {
    // Implementation
    return &Resp{sign: SimpleString, str: "OK"}
}
```

2. Register in `Handlers` map:
```go
var Handlers = map[string]Handler{
    // ...
    "MYCOMMAND": mycommand,
}
```

## License

This project is a learning implementation of Redis functionality in Go.

## Contributing

This appears to be an educational project. Contributions and improvements are welcome!

## Acknowledgments

Inspired by Redis (https://redis.io) and implements the RESP protocol specification.

