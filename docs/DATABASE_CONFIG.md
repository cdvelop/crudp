# Database Configuration for SSE Broker

## Overview

SSE Broker supports optional persistence for request queues using a simple key-value store interface. This allows queues to survive client refreshes (WASM) or server restarts. The user provides the implementation, enabling flexibility for different storage backends.

## KVStore Interface

**File: `interfaces.go`**
```go
// KVStore provides simple key-value persistence for queue
type KVStore interface {
    Get(key string) (string, error)
    Set(key, value string) error
}
```

## Example Implementations

### WASM Client (Browser localStorage)

```go
//go:build wasm
package main

import (
    "syscall/js"
)

type LocalStorageStore struct{}

func (ls *LocalStorageStore) Get(key string) (string, error) {
    val := js.Global().Get("localStorage").Call("getItem", key)
    if val.IsNull() {
        return "", fmt.Errorf("key not found")
    }
    return val.String(), nil
}

func (ls *LocalStorageStore) Set(key, value string) error {
    js.Global().Get("localStorage").Call("setItem", key, value)
    return nil
}
```

### Server (In-Memory Map)

```go
//go:build !wasm
package main

import "sync"

type MemoryStore struct {
    mu   sync.RWMutex
    data map[string]string
}

func NewMemoryStore() *MemoryStore {
    return &MemoryStore{
        data: make(map[string]string),
    }
}

func (m *MemoryStore) Get(key string) (string, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    val, ok := m.data[key]
    if !ok {
        return "", fmt.Errorf("key not found")
    }
    return val, nil
}

func (m *MemoryStore) Set(key, value string) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.data[key] = value
    return nil
}
```

### Server (External Database - Redis Example)

```go
//go:build !wasm
package main

import (
    "context"
    "github.com/redis/go-redis/v9"
)

type RedisStore struct {
    client *redis.Client
}

func NewRedisStore(addr string) *RedisStore {
    rdb := redis.NewClient(&redis.Options{
        Addr: addr,
    })
    return &RedisStore{client: rdb}
}

func (r *RedisStore) Get(key string) (string, error) {
    val, err := r.client.Get(context.Background(), key).Result()
    if err == redis.Nil {
        return "", fmt.Errorf("key not found")
    }
    return val, err
}

func (r *RedisStore) Set(key, value string) error {
    return r.client.Set(context.Background(), key, value, 0).Err()
}
```

## Configuration

### Injecting KVStore into BrokerConfig

**File: `config.go`**
```go
config := crudp.DefaultConfig()
config.Store = &MemoryStore{} // or &RedisStore{...} or &LocalStorageStore{}
cp := crudp.New(config)
```

### Queue Persistence Behavior

- **Client (WASM):** Queues persist in localStorage, survive page refreshes
- **Server:** Queues persist based on implementation (memory = lost on restart, Redis/DB = survive)
- **Serialization:** Queues are stored as JSON strings containing `[]QueuedPacket`


## Related Documentation

- [SSE_BROKER.md](SSE_BROKER.md) - Main SSE Broker documentation
- [INTEGRATION_GUIDE.md](INTEGRATION_GUIDE.md) - Integration examples