package gamequery

import (
	"context"
	"sync"
	"time"
)

// Cache answers from a copy no older than its TTL.
//
// This is not about the cost of the query — it is one UDP round trip. It is
// about the game server's own rate limit: srcds drops queries from a source that
// asks more than a few times a second (sv_max_queries_sec, over a 30 second
// window), and it keeps dropping them until that window drains. Without a cache,
// anything that scrapes in a burst — a person with curl, a second Prometheus, a
// short scrape interval — makes the server stop answering and the dashboard read
// "not answering" for half a minute. Measured, not hypothetical.
//
// The TTL is the query rate: one query per TTL, whatever asks.
type Cache struct {
	addr    string
	ttl     time.Duration
	timeout time.Duration

	// mu is held across the query on purpose: concurrent scrapes then share one
	// answer instead of each sending its own packet, which is the whole point.
	mu   sync.Mutex
	at   time.Time
	info Info
	err  error
}

// NewCache returns a cache over one address. timeout bounds a single query.
func NewCache(addr string, ttl, timeout time.Duration) *Cache {
	return &Cache{addr: addr, ttl: ttl, timeout: timeout}
}

// Info returns what the server last said, querying it when that is too old. A
// failure is cached like an answer: retrying it on every scrape is exactly the
// burst the rate limit punishes.
func (c *Cache) Info(ctx context.Context) (Info, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.at.IsZero() && time.Since(c.at) < c.ttl {
		return c.info, c.err
	}
	c.info, c.err = Query(ctx, c.addr, c.timeout)
	c.at = time.Now()
	return c.info, c.err
}
