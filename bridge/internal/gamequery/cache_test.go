package gamequery

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// countingServer answers every request and counts how many it was sent, which is
// the number the game server's rate limiter would be counting too.
func countingServer(t *testing.T, players byte) (string, *atomic.Int64) {
	t.Helper()
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	var asked atomic.Int64
	reply := infoReply(players, 6, 0)
	go func() {
		buffer := make([]byte, 1024)
		for {
			_, from, err := conn.ReadFrom(buffer)
			if err != nil {
				return
			}
			asked.Add(1)
			if _, err := conn.WriteTo(reply, from); err != nil {
				return
			}
		}
	}()
	return conn.LocalAddr().String(), &asked
}

func TestCacheAsksOncePerTTL(t *testing.T) {
	addr, asked := countingServer(t, 3)
	cache := NewCache(addr, 200*time.Millisecond, time.Second)

	for range 5 {
		got, err := cache.Info(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if got.Players != 3 {
			t.Fatalf("players = %d", got.Players)
		}
	}
	if n := asked.Load(); n != 1 {
		t.Fatalf("the server was asked %d times for five scrapes", n)
	}
}

func TestCacheAsksAgainOnceStale(t *testing.T) {
	addr, asked := countingServer(t, 1)
	cache := NewCache(addr, 20*time.Millisecond, time.Second)

	if _, err := cache.Info(context.Background()); err != nil {
		t.Fatal(err)
	}
	time.Sleep(40 * time.Millisecond)
	if _, err := cache.Info(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := asked.Load(); n != 2 {
		t.Fatalf("the server was asked %d times, want 2", n)
	}
}

func TestCacheSharesOneQueryBetweenConcurrentScrapes(t *testing.T) {
	addr, asked := countingServer(t, 2)
	cache := NewCache(addr, time.Minute, time.Second)

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			if _, err := cache.Info(context.Background()); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if n := asked.Load(); n != 1 {
		t.Fatalf("the server was asked %d times by eight concurrent scrapes", n)
	}
}

func TestCacheHoldsAFailureToo(t *testing.T) {
	// Retrying a dead server on every scrape is the same burst the rate limit
	// punishes, so a failure is held for the TTL like an answer.
	cache := NewCache("127.0.0.1:1", time.Minute, 50*time.Millisecond)

	start := time.Now()
	if _, err := cache.Info(context.Background()); err == nil {
		t.Fatal("a closed port answered")
	}
	if _, err := cache.Info(context.Background()); err == nil {
		t.Fatal("a closed port answered")
	}
	// Two timeouts would take twice the timeout; one cached failure does not.
	if elapsed := time.Since(start); elapsed > 150*time.Millisecond {
		t.Fatalf("two calls took %s, so the failure was not cached", elapsed)
	}
}
