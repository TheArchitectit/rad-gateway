// Package cache provides Redis-backed caching for RAD Gateway.
package cache

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// RedisCache implements Cache interface using Redis protocol.
// This is a lightweight implementation that speaks the Redis protocol
// directly without requiring external dependencies.
type RedisCache struct {
	addr      string
	password  string
	database  int
	keyPrefix string
	conn      net.Conn
	reader    *redisReader
}

// redisReader provides buffered reading from Redis connection.
type redisReader struct {
	conn   net.Conn
	buffer []byte
	offset int
}

// NewRedis creates a new Redis cache client.
func NewRedis(config Config) (*RedisCache, error) {
	addr := config.Address
	if addr == "" {
		addr = "localhost:6379"
	}

	// Establish TCP connection
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	cache := &RedisCache{
		addr:      addr,
		password:  config.Password,
		database:  config.Database,
		keyPrefix: config.KeyPrefix,
		conn:      conn,
		reader:    &redisReader{conn: conn, buffer: make([]byte, 4096)},
	}

	// Authenticate if password provided
	if config.Password != "" {
		if err := cache.auth(context.Background()); err != nil {
			conn.Close()
			return nil, fmt.Errorf("redis authentication failed: %w", err)
		}
	}

	// Select database
	if config.Database > 0 {
		if err := cache.selectDB(context.Background(), config.Database); err != nil {
			conn.Close()
			return nil, fmt.Errorf("redis select database failed: %w", err)
		}
	}

	return cache, nil
}

// auth sends AUTH command to Redis.
func (r *RedisCache) auth(ctx context.Context) error {
	cmd := fmt.Sprintf("*2\r\n$4\r\nAUTH\r\n$%d\r\n%s\r\n", len(r.password), r.password)
	if err := r.sendCommand(ctx, cmd); err != nil {
		return err
	}
	return r.readSimpleString()
}

// selectDB sends SELECT command to Redis.
func (r *RedisCache) selectDB(ctx context.Context, db int) error {
	cmd := fmt.Sprintf("*2\r\n$6\r\nSELECT\r\n$%d\r\n%d\r\n", len(fmt.Sprintf("%d", db)), db)
	if err := r.sendCommand(ctx, cmd); err != nil {
		return err
	}
	return r.readSimpleString()
}

// sendCommand sends a raw Redis command.
func (r *RedisCache) sendCommand(ctx context.Context, cmd string) error {
	deadline, ok := ctx.Deadline()
	if ok {
		r.conn.SetWriteDeadline(deadline)
		defer r.conn.SetWriteDeadline(time.Time{})
	}
	_, err := r.conn.Write([]byte(cmd))
	return err
}

// readSimpleString reads a simple string response (+OK\r\n).
func (r *RedisCache) readSimpleString() error {
	line, err := r.readLine()
	if err != nil {
		return err
	}
	if len(line) == 0 {
		return errors.New("empty response")
	}
	if line[0] == '-' {
		return fmt.Errorf("redis error: %s", line[1:])
	}
	if line[0] != '+' {
		return fmt.Errorf("unexpected response: %s", line)
	}
	return nil
}

// readBulkString reads a bulk string response ($<length>\r\n<data>\r\n).
func (r *RedisCache) readBulkString() ([]byte, error) {
	line, err := r.readLine()
	if err != nil {
		return nil, err
	}
	if len(line) == 0 {
		return nil, errors.New("empty response")
	}
	if line[0] == '-' {
		return nil, fmt.Errorf("redis error: %s", line[1:])
	}
	if line[0] == '$' {
		if line == "$-1" {
			return nil, nil // Key not found
		}
		var length int
		if _, err := fmt.Sscanf(line, "$%d", &length); err != nil {
			return nil, fmt.Errorf("invalid bulk string length: %w", err)
		}
		data := make([]byte, length+2) // +2 for \r\n
		if err := r.readFull(data); err != nil {
			return nil, err
		}
		return data[:length], nil
	}
	return nil, fmt.Errorf("unexpected response type: %s", line)
}

// readInteger reads an integer response (:<number>\r\n).
func (r *RedisCache) readInteger() (int64, error) {
	line, err := r.readLine()
	if err != nil {
		return 0, err
	}
	if len(line) == 0 {
		return 0, errors.New("empty response")
	}
	if line[0] == '-' {
		return 0, fmt.Errorf("redis error: %s", line[1:])
	}
	if line[0] == ':' {
		var val int64
		if _, err := fmt.Sscanf(line, ":%d", &val); err != nil {
			return 0, fmt.Errorf("invalid integer: %w", err)
		}
		return val, nil
	}
	return 0, fmt.Errorf("unexpected response type: %s", line)
}

// readLine reads a single line ending with \r\n.
func (r *RedisCache) readLine() (string, error) {
	var result []byte
	for {
		if r.reader.offset >= len(r.reader.buffer) {
			n, err := r.conn.Read(r.reader.buffer)
			if err != nil {
				return "", err
			}
			r.reader.offset = 0
			r.reader.buffer = r.reader.buffer[:n]
		}

		for i := r.reader.offset; i < len(r.reader.buffer); i++ {
			if i+1 < len(r.reader.buffer) && r.reader.buffer[i] == '\r' && r.reader.buffer[i+1] == '\n' {
				result = append(result, r.reader.buffer[r.reader.offset:i]...)
				r.reader.offset = i + 2
				return string(result), nil
			}
		}

		result = append(result, r.reader.buffer[r.reader.offset:]...)
		r.reader.offset = len(r.reader.buffer)
	}
}

// readFull reads exactly n bytes.
func (r *RedisCache) readFull(buf []byte) error {
	total := 0
	for total < len(buf) {
		n, err := r.conn.Read(buf[total:])
		if err != nil {
			return err
		}
		total += n
	}
	return nil
}

// prefixKey adds the configured prefix to a key.
func (r *RedisCache) prefixKey(key string) string {
	return r.keyPrefix + key
}

// Get retrieves a value from Redis.
func (r *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	prefixedKey := r.prefixKey(key)
	cmd := fmt.Sprintf("*2\r\n$3\r\nGET\r\n$%d\r\n%s\r\n", len(prefixedKey), prefixedKey)

	deadline, ok := ctx.Deadline()
	if ok {
		r.conn.SetDeadline(deadline)
		defer r.conn.SetDeadline(time.Time{})
	}

	if err := r.sendCommand(ctx, cmd); err != nil {
		return nil, err
	}

	return r.readBulkString()
}

// Set stores a value in Redis with TTL.
func (r *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	prefixedKey := r.prefixKey(key)
	seconds := int(ttl.Seconds())

	var cmd string
	if seconds > 0 {
		cmd = fmt.Sprintf("*5\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n$2\r\nEX\r\n$%d\r\n%d\r\n",
			len(prefixedKey), prefixedKey,
			len(value), value,
			len(fmt.Sprintf("%d", seconds)), seconds)
	} else {
		cmd = fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n",
			len(prefixedKey), prefixedKey,
			len(value), value)
	}

	deadline, ok := ctx.Deadline()
	if ok {
		r.conn.SetDeadline(deadline)
		defer r.conn.SetDeadline(time.Time{})
	}

	if err := r.sendCommand(ctx, cmd); err != nil {
		return err
	}

	return r.readSimpleString()
}

// Delete removes a key from Redis.
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	prefixedKey := r.prefixKey(key)
	cmd := fmt.Sprintf("*2\r\n$3\r\nDEL\r\n$%d\r\n%s\r\n", len(prefixedKey), prefixedKey)

	deadline, ok := ctx.Deadline()
	if ok {
		r.conn.SetDeadline(deadline)
		defer r.conn.SetDeadline(time.Time{})
	}

	if err := r.sendCommand(ctx, cmd); err != nil {
		return err
	}

	_, err := r.readInteger()
	return err
}

// DeletePattern removes all keys matching a pattern using SCAN + DEL.
func (r *RedisCache) DeletePattern(ctx context.Context, pattern string) error {
	// For simplicity, we'll iterate with SCAN
	prefixedPattern := r.prefixKey(pattern)
	cursor := "0"

	for {
		cmd := fmt.Sprintf("*6\r\n$4\r\nSCAN\r\n$%d\r\n%s\r\n$5\r\nMATCH\r\n$%d\r\n%s\r\n$5\r\nCOUNT\r\n$3\r\n100\r\n",
			len(cursor), cursor,
			len(prefixedPattern), prefixedPattern)

		deadline, ok := ctx.Deadline()
		if ok {
			r.conn.SetDeadline(deadline)
			defer r.conn.SetDeadline(time.Time{})
		}

		if err := r.sendCommand(ctx, cmd); err != nil {
			return err
		}

		// Read array response
		keys, nextCursor, err := r.readScanResponse()
		if err != nil {
			return err
		}

		// Delete keys
		for _, key := range keys {
			// Remove prefix for Delete method
			unprefixedKey := strings.TrimPrefix(key, r.keyPrefix)
			if err := r.Delete(ctx, unprefixedKey); err != nil {
				return err
			}
		}

		if nextCursor == "0" {
			break
		}
		cursor = nextCursor
	}

	return nil
}

// readScanResponse reads SCAN command response.
func (r *RedisCache) readScanResponse() ([]string, string, error) {
	line, err := r.readLine()
	if err != nil {
		return nil, "", err
	}

	if line[0] != '*' {
		return nil, "", fmt.Errorf("expected array, got: %s", line)
	}

	var count int
	fmt.Sscanf(line, "*%d", &count)

	// Read cursor (first element)
	cursorLine, err := r.readLine()
	if err != nil {
		return nil, "", err
	}

	var cursor string
	if cursorLine[0] == '$' {
		var length int
		fmt.Sscanf(cursorLine, "$%d", &length)
		cursorBytes := make([]byte, length+2)
		r.readFull(cursorBytes)
		cursor = string(cursorBytes[:length])
	}

	// Read keys array (second element)
	keysLine, err := r.readLine()
	if err != nil {
		return nil, "", err
	}

	var keys []string
	if keysLine[0] == '*' {
		var keyCount int
		fmt.Sscanf(keysLine, "*%d", &keyCount)

		for i := 0; i < keyCount; i++ {
			keyLine, _ := r.readLine()
			if keyLine[0] == '$' {
				var length int
				fmt.Sscanf(keyLine, "$%d", &length)
				keyBytes := make([]byte, length+2)
				r.readFull(keyBytes)
				keys = append(keys, string(keyBytes[:length]))
			}
		}
	}

	return keys, cursor, nil
}

// Ping checks the Redis connection.
func (r *RedisCache) Ping(ctx context.Context) error {
	deadline, ok := ctx.Deadline()
	if ok {
		r.conn.SetDeadline(deadline)
		defer r.conn.SetDeadline(time.Time{})
	}

	if err := r.sendCommand(ctx, "*1\r\n$4\r\nPING\r\n"); err != nil {
		return err
	}

	line, err := r.readLine()
	if err != nil {
		return err
	}
	if line != "+PONG" {
		return fmt.Errorf("unexpected ping response: %s", line)
	}
	return nil
}

// Close closes the Redis connection.
func (r *RedisCache) Close() error {
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

// Ensure RedisCache implements Cache interface.
var _ Cache = (*RedisCache)(nil)
