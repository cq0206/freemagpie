// freemagpie // The catalog (API.md §1): the anonymous, public list of
// free-credit offers. It is cached on disk so the Free Credits Market
// page opens with the cloud unreachable and off (§1.2.1: not signing
// in, and no network, still browses). A read that fails answers the
// cache and its age, never an empty catalog passed off as fresh.
package freemagpie

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/yetone/magpie/internal/appdir"
	"github.com/yetone/magpie/internal/filelock"
)

// cacheFile is where the catalog last read is kept, beside magpie's
// other files (a name of its own, never settings.json or providers.json).
func cacheFile() string { return appdir.Join(appdir.Config(), "freemagpie-cache.json") }

// catalogCache is the cached catalog and when it was read, kept in
// memory beside the file so the page reads it once a process.
type catalogCache struct {
	mu     sync.Mutex
	offers []Offer
	readAt time.Time
	loaded bool
}

// Cache is the catalog's cache, read by the GUI (its age is what the
// page tells the reader: "read 2 days ago").
var Cache catalogCache

// cacheDisk is the file's shape: what was read, and when.
type cacheDisk struct {
	Offers []Offer   `json:"offers"`
	ReadAt time.Time `json:"read_at"`
}

// readCache loads the file once a process; a file that can't be read
// or doesn't parse is no cache (unknown), not an empty catalog.
func (c *catalogCache) readCache() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.loaded {
		return
	}
	c.loaded = true
	b, err := os.ReadFile(cacheFile())
	if err != nil {
		return
	}
	var d cacheDisk
	if json.Unmarshal(b, &d) != nil || d.ReadAt.IsZero() {
		return
	}
	c.offers, c.readAt = d.Offers, d.ReadAt
}

// writeCache keeps what was read, whole-file under a lock against the
// other magpies (settings' own pattern: temp, sync, rename).
func (c *catalogCache) writeCache(offers []Offer) {
	now := time.Now().UTC()
	b, err := json.Marshal(cacheDisk{Offers: offers, ReadAt: now})
	if err != nil {
		return
	}
	unlock, _ := filelock.Lock(cacheFile()+".lock", filelock.Wait)
	defer unlock()
	tmp := cacheFile() + ".tmp"
	if os.WriteFile(tmp, b, 0o600) == nil {
		_ = os.Rename(tmp, cacheFile())
	}
	c.mu.Lock()
	c.offers, c.readAt, c.loaded = offers, now, true
	c.mu.Unlock()
}

// Cached is the catalog as last read (nil, and a zero time, when none
// was), for a page that opens with the cloud off or unreachable.
func (c *catalogCache) Cached() (offers []Offer, readAt time.Time) {
	c.readCache()
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.offers, c.readAt
}

// ResetForTest forgets the cache, so a test reads the file as a fresh
// process would. Only for tests. The struct stays: its mutex is
// unlocked after this, so replacing it whole would leave the deferred
// unlock on a fresh, unlocked one.
func (c *catalogCache) ResetForTest() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.offers, c.readAt, c.loaded = nil, time.Time{}, false
}

// OffersQuery picks the offers asked for (API.md §1): every field
// optional, "" and nil meaning no pick.
type OffersQuery struct {
	Product     string
	Category    string
	RewardKind  string
	Region      string
	APIOnly     bool
	ResetPeriod string
	Sort        string // recent (the default) | verified | expiring | popular
	Limit       int
}

// Offers reads the catalog (anonymous, no credentials — §1.2.1), walks
// its pages up to a sane bound, and keeps what was read as the cache.
// It answers the offers and when they were read; an error means the
// cloud was asked and didn't answer, with the cache still readable by
// Cached.
func Offers(ctx context.Context, q OffersQuery) ([]Offer, time.Time, error) {
	var all []Offer
	cursor := ""
	for page := 0; page < 10; page++ {
		var got CatalogPage
		path := offersPath(q, cursor)
		if err := get(ctx, path, &got); err != nil {
			return nil, time.Time{}, err
		}
		all = append(all, got.Data...)
		if got.NextCursor == nil || *got.NextCursor == "" || (q.Limit > 0 && len(all) >= q.Limit) {
			break
		}
		cursor = *got.NextCursor
	}
	if len(all) > 0 {
		Cache.writeCache(all)
	}
	return all, time.Now(), nil
}

// offersPath builds /api/v1/offers?… from the query (API.md §1).
func offersPath(q OffersQuery, cursor string) string {
	v := url.Values{}
	add := func(k, s string) {
		if s != "" {
			v.Set(k, s)
		}
	}
	add("product", q.Product)
	add("category", q.Category)
	add("reward_kind", q.RewardKind)
	add("region", q.Region)
	add("reset_period", q.ResetPeriod)
	add("sort", q.Sort)
	if q.APIOnly {
		v.Set("api_compatible", "true")
	}
	if q.Limit > 0 {
		v.Set("limit", strconv.Itoa(q.Limit))
	}
	if cursor != "" {
		v.Set("cursor", cursor)
	}
	p := "/api/v1/offers"
	if s := v.Encode(); s != "" {
		p += "?" + s
	}
	return p
}
