// freemagpie // Package freemagpie is the desktop client of FreeMagpie
// Cloud (docs/freemagpie/API.md): the free-credits catalog, claim
// tracking and — only when the user turns it on — sanitized quota
// snapshots. It is a client only: it never touches the gateway, and
// every call can fail (the catalog answers a cache, the rest an error)
// so the app works with the cloud unreachable or not signed in
// (ARCHITECTURE.md §1.2.1: not signing in buys sync, not use).
package freemagpie

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/yetone/magpie/internal/netproxy"
	"github.com/yetone/magpie/internal/settings"
)

// DefaultURL is the cloud the desktop talks to unless the user set
// another in Settings (settings.FreeMagpieURL).
const DefaultURL = "https://api.freemagpie.com"

// ErrNoCloud is what a call to the cloud answers when the user has
// none configured: the caller shows its cache or says so, never an
// empty catalog passed off as the real one (a failed read is unknown).
var ErrNoCloud = errors.New("FreeMagpie Cloud is off")

// Offer is one entry of the catalog (API.md §1): a free-credit
// opportunity of a product. A native credit is a use-it-in-their-app
// reward; only an API credit can be added to magpie.
type Offer struct {
	ID                 string     `json:"id"`
	Slug               string     `json:"slug"`
	Product            Product    `json:"product"`
	Title              string     `json:"title"`
	RewardKind         string     `json:"reward_kind"` // native_credit | api_credit | free_model | promo
	RewardLabel        string     `json:"reward_label"`
	ResetPeriod        string     `json:"reset_period"` // daily | weekly | monthly | none
	ClaimMethod        string     `json:"claim_method"` // manual | api | deeplink
	ClaimURL           string     `json:"claim_url"`
	RegionCodes        []string   `json:"region_codes"`
	ValidUntil         *time.Time `json:"valid_until"`
	APICompatible      bool       `json:"api_compatible"`
	VerificationStatus string     `json:"verification_status"` // official_verified | community_verified | unverified | expired
	LastVerifiedAt     *time.Time `json:"last_verified_at"`
	ImportLink         string     `json:"import_link,omitempty"` // magpie://import?… (API.md §3), an API offer only
}

// Product is what an offer is of (API.md §1): an agent, a provider, a
// model or a tool, with its own directory page.
type Product struct {
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Icon     string `json:"icon,omitempty"`
	Category string `json:"category"` // coding-agent | general-agent | api-provider | model
}

// CatalogPage is one page of the catalog (API.md §0): data plus the
// cursor the next page asks for, nil when this was the last.
type CatalogPage struct {
	Data       []Offer `json:"data"`
	NextCursor *string `json:"next_cursor"`
}

// OfferState is one offer's state of the user's own (API.md §4): saved
// to come back to, claimed once done, dismissed out of the way.
type OfferState struct {
	OfferSlug string     `json:"offer_slug"`
	State     string     `json:"state"` // saved | claimed | dismissed
	ClaimedAt *time.Time `json:"claimed_at,omitempty"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// CloudURL is the cloud the user has, "" when they use none.
func CloudURL() string {
	if u := strings.TrimSpace(settings.Load().FreeMagpieURL); u != "" {
		return strings.TrimRight(u, "/")
	}
	return DefaultURL
}

// cloudOn is whether the user has a cloud to ask: a URL of their own,
// or the default one — off is a URL set to "off", which no catalog
// call passes.
func cloudOn() bool { return CloudURL() != "off" }

// client is the HTTP client every call shares: magpie's own proxy
// settings, as its other cloud-bound requests take them (otlp, update).
var client = &http.Client{
	Timeout:   15 * time.Second,
	Transport: &http.Transport{Proxy: netproxy.Func, MaxIdleConnsPerHost: 2, IdleConnTimeout: 90 * time.Second},
}

// get reads the cloud once (API.md §0: a request the desktop makes
// must be able to fail — one try, no retry loop to hang a page on).
func get(ctx context.Context, path string, out any) error {
	if !cloudOn() {
		return ErrNoCloud
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, CloudURL()+path, nil)
	if err != nil {
		return err
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		var e struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(body, &e) == nil && e.Error.Message != "" {
			return fmt.Errorf("%s: %s", e.Error.Code, e.Error.Message)
		}
		return fmt.Errorf("the cloud answered %s", res.Status)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("the cloud's answer is not the API.md shape: %w", err)
	}
	return nil
}
