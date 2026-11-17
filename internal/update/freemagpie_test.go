package update

import "testing"

// freemagpie: a release must stay on its distribution's update channel,
// while explicit user/test environment overrides continue to win.
func TestDistributionFeeds(t *testing.T) {
	oldFeed, oldNotes, oldPage := defaultFeed, defaultNotesFeed, releasePagePrefix
	t.Cleanup(func() {
		defaultFeed, defaultNotesFeed, releasePagePrefix = oldFeed, oldNotes, oldPage
	})
	defaultFeed = "https://freemagpie.com/api/latest"
	defaultNotesFeed = "https://freemagpie.com/api/notes"
	releasePagePrefix = "https://github.com/cq0206/freemagpie/releases/tag/freemagpie-v"
	t.Setenv("MAGPIE_UPDATE_FEED", "")
	t.Setenv("MAGPIE_NOTES_FEED", "")
	if got := Feed(); got != defaultFeed {
		t.Fatalf("distribution update feed = %q", got)
	}
	if got := NotesFeed(); got != defaultNotesFeed {
		t.Fatalf("distribution notes feed = %q", got)
	}
	if got := ReleasePage(" v0.1.0 "); got != releasePagePrefix+"0.1.0" {
		t.Fatalf("distribution release page = %q", got)
	}
	t.Setenv("MAGPIE_UPDATE_FEED", "https://example.test/latest")
	t.Setenv("MAGPIE_NOTES_FEED", "https://example.test/notes")
	if Feed() != "https://example.test/latest" || NotesFeed() != "https://example.test/notes" {
		t.Fatal("explicit feed overrides were ignored")
	}
	if Site != "https://usemagpie.ai" {
		t.Fatal("non-update upstream services must keep their address")
	}
}
