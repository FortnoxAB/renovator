package repoowner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeConfig(t *testing.T, baseDir, platform, repo, body string) {
	t.Helper()
	dir := filepath.Join(baseDir, "repos", platform, filepath.FromSlash(repo))
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestWatch(t *testing.T) {
	base := t.TempDir()
	platform := "bitbucket-server"
	r := &Resolver{BaseDir: base, Platform: platform, PollInterval: time.Millisecond}

	writeConfig(t, base, platform, "FNX/currencies", `{"serviceName":"currencies","team":"gonix"}`)
	writeConfig(t, base, platform, "GL/no-team", `{"serviceName":"cache"}`)
	writeConfig(t, base, platform, "GL/broken", `{not json`)

	cases := []struct {
		name       string
		repo       string
		wantTeam   string
		wantSvcNam string
	}{
		{"full config", "FNX/currencies", "gonix", "currencies"},
		{"strips options", "FNX/currencies?loglevel=debug", "gonix", "currencies"},
		{"missing team falls back, keeps serviceName", "GL/no-team", Unknown, "cache"},
		{"broken json falls back to slug", "GL/broken", Unknown, "GL/broken"},
		{"missing file falls back to slug", "GL/does-not-exist", Unknown, "GL/does-not-exist"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			team, name := r.Watch(context.Background(), tc.repo).Result()
			if team != tc.wantTeam {
				t.Errorf("team = %q, want %q", team, tc.wantTeam)
			}
			if name != tc.wantSvcNam {
				t.Errorf("name = %q, want %q", name, tc.wantSvcNam)
			}
		})
	}
}

func TestWatchPicksUpCloneCreatedAfterStart(t *testing.T) {
	base := t.TempDir()
	platform := "bitbucket-server"
	r := &Resolver{BaseDir: base, Platform: platform, PollInterval: time.Millisecond}

	watch := r.Watch(context.Background(), "FO/wzrd")
	time.Sleep(20 * time.Millisecond) // first look must miss, ie not cloned yet
	writeConfig(t, base, platform, "FO/wzrd", `{"serviceName":"database-portal","team":"wizards"}`)
	time.Sleep(100 * time.Millisecond)

	team, name := watch.Result()
	if team != "wizards" || name != "database-portal" {
		t.Errorf("team, name = %q, %q, want %q, %q", team, name, "wizards", "database-portal")
	}
}
