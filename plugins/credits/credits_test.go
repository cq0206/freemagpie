package credits

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yetone/magpie/internal/freemagpie"
	"github.com/yetone/magpie/internal/pluginrt"
	"github.com/yetone/magpie/internal/settings"
	"github.com/yetone/magpie/internal/testenv"
)

// The tests run in a home of their own: the offer states are a file
// beside magpie's others, and no test touches a real one.
func TestMain(m *testing.M) { testenv.Main(m) }

// TestCreditsRegistered: the plugin is there, with its page in the
// window and its client files on disk (TestBuiltinClientsLoaded holds
// the page to loading them; this holds the manifest's own say).
func TestCreditsRegistered(t *testing.T) {
	var found *pluginrt.Manifest
	for _, m := range pluginrt.Registered() {
		if m.ID == "credits" {
			found = &m
		}
	}
	if found == nil {
		t.Fatal("the credits plugin isn't registered")
	}
	var page bool
	for _, p := range found.Parts {
		if p.Slot == "nav.page" && p.Key == "credits" {
			page = true
		}
	}
	if !page {
		t.Fatalf("no nav.page part: %+v", found.Parts)
	}
	if found.InUse == nil {
		t.Fatal("no InUse: the page could be switched off with offers saved")
	}
}

// TestInUseCountsSavedOffers: the states the user keeps hold the plugin
// on, and an unreadable states file keeps it on too (unknown is in
// use, not unused).
func TestInUseCountsSavedOffers(t *testing.T) {
	m := pluginrt.Registered()
	var found *pluginrt.Manifest
	for i := range m {
		if m[i].ID == "credits" {
			found = &m[i]
		}
	}
	if found == nil {
		t.Fatal("not registered")
	}
	n, err := found.InUse()
	if err != nil || n != 0 {
		t.Fatalf("nothing saved: %d, %v", n, err)
	}
	if err := freemagpie.SetState("qoder-daily", "saved", nil); err != nil {
		t.Fatal(err)
	}
	n, err = found.InUse()
	if err != nil || n != 1 {
		t.Fatalf("one saved: %d, %v", n, err)
	}

	// an unreadable file is an error, and an error is in use
	if err := os.WriteFile(filepath.Join(settings.Dir(), "freemagpie.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	freemagpie.ResetStatesForTest()
	if _, err := found.InUse(); err == nil {
		t.Fatal("a broken states file reads as fine")
	}
}
