// Package store provides the small amount of Redis state the worker needs.
package store

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// RedisDeduplicator records delivered listing IDs in Redis.
type RedisDeduplicator struct {
	address string
}

func NewRedisDeduplicator(address string) *RedisDeduplicator {
	return &RedisDeduplicator{address: address}
}

// MarkDelivered records an item for ttl. It returns true if it was not already recorded.
func (d *RedisDeduplicator) MarkDelivered(ctx context.Context, itemID string, ttl time.Duration) (bool, error) {
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", d.address)
	if err != nil {
		return false, fmt.Errorf("connect to Redis: %w", err)
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	}

	key := "vinted:delivered:" + itemID
	command := encodeCommand("SET", key, "1", "NX", "PX", strconv.FormatInt(ttl.Milliseconds(), 10))
	if _, err := conn.Write(command); err != nil {
		return false, fmt.Errorf("write Redis command: %w", err)
	}

	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("read Redis response: %w", err)
	}
	switch strings.TrimSpace(line) {
	case "+OK":
		return true, nil
	case "$-1":
		return false, nil
	default:
		return false, fmt.Errorf("unexpected Redis response: %q", strings.TrimSpace(line))
	}
}

// Forget removes a previously claimed item so a failed delivery can be retried.
func (d *RedisDeduplicator) Forget(ctx context.Context, itemID string) error {
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", d.address)
	if err != nil {
		return fmt.Errorf("connect to Redis: %w", err)
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	}
	if _, err := conn.Write(encodeCommand("DEL", "vinted:delivered:"+itemID)); err != nil {
		return fmt.Errorf("write Redis command: %w", err)
	}
	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return fmt.Errorf("read Redis response: %w", err)
	}
	if !strings.HasPrefix(line, ":") {
		return fmt.Errorf("unexpected Redis response: %q", strings.TrimSpace(line))
	}
	return nil
}

func encodeCommand(values ...string) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "*%d\r\n", len(values))
	for _, value := range values {
		fmt.Fprintf(&b, "$%d\r\n%s\r\n", len(value), value)
	}
	return []byte(b.String())
}
