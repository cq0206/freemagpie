package appdir

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func freeHome(t *testing.T) string {
	t.Helper()
	old := Name
	Name = "freemagpie"
	UseExecutable("")
	t.Cleanup(func() { Name = old; UseExecutable("") })
	h := t.TempDir()
	for _, v := range []string{"HOME", "USERPROFILE"} {
		t.Setenv(v, h)
	}
	for _, v := range []string{"XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME", "LOCALAPPDATA", "APPDATA"} {
		t.Setenv(v, "")
	}
	return h
}

func TestFreeMagpieOwnDirectories(t *testing.T) {
	h := freeHome(t)
	for _, item := range []struct{ got, want string }{
		{Config(), filepath.Join(h, ".config", "freemagpie")},
		{Cache(), filepath.Join(h, ".cache", "freemagpie")},
		{logsOn("darwin"), filepath.Join(h, "Library", "Logs", "freemagpie")},
		{logsOn("windows"), filepath.Join(h, "AppData", "Local", "freemagpie", "logs")},
		{logsOn("linux"), filepath.Join(h, ".local", "state", "freemagpie")},
	} {
		if item.got != item.want {
			t.Errorf("got %q, want %q", item.got, item.want)
		}
	}
	if cache, err := SystemCache(); err != nil || filepath.Base(cache) != "freemagpie" {
		t.Fatalf("system cache: %q %v", cache, err)
	}
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	if Config() != filepath.Join(root, "freemagpie") {
		t.Fatal(Config())
	}
	if Redirected() != filepath.Join(h, ".config", "freemagpie") {
		t.Fatal(Redirected())
	}
	if runtime.GOOS == "windows" && WebView() != filepath.Join(Config(), "webview2") {
		t.Fatal(WebView())
	}
}

func TestFreeMagpieImportKeepsSourceAndExistingProfile(t *testing.T) {
	h := freeHome(t)
	src := filepath.Join(h, ".config", "magpie")
	if err := os.MkdirAll(src, 0o700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(src, "providers.json")
	if err := os.WriteFile(file, []byte(`{"source":"old"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ImportLegacyConfig(); err != nil {
		t.Fatal(err)
	}
	old, _ := os.ReadFile(file)
	got, _ := os.ReadFile(filepath.Join(Config(), "providers.json"))
	if string(got) != string(old) {
		t.Fatal("profile not copied")
	}
	if err := os.WriteFile(filepath.Join(Config(), "settings.json"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ImportLegacyConfig(); err == nil {
		t.Fatal("existing profile accepted")
	}
	got, _ = os.ReadFile(filepath.Join(Config(), "settings.json"))
	if string(got) != "new" {
		t.Fatal("existing profile changed")
	}
}

func TestFreeMagpieImportRefusesLinksAndPortable(t *testing.T) {
	h := freeHome(t)
	src := filepath.Join(h, ".config", "magpie")
	if err := os.MkdirAll(src, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(src, "linked")); err != nil {
		t.Skip(err)
	}
	if err := ImportLegacyConfig(); err == nil {
		t.Fatal("linked profile imported")
	}
	if _, err := os.Lstat(Config()); !os.IsNotExist(err) {
		t.Fatal("failed import published destination")
	}
	portable = filepath.Join(t.TempDir(), "data")
	if err := ImportLegacyConfig(); err == nil {
		t.Fatal("portable profile imported")
	}
}
