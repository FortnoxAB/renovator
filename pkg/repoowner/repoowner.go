package repoowner

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const Unknown = "unknown"

const defaultPollInterval = 200 * time.Millisecond

type repoConfig struct {
	ServiceName string `json:"serviceName"`
	Team        string `json:"team"`
}

type Resolver struct {
	BaseDir      string
	Platform     string
	PollInterval time.Duration
}

// NewFromEnv mirrors renovate's baseDir resolution in lib/workers/global/initialize.ts.
func NewFromEnv() *Resolver {
	baseDir := os.Getenv("RENOVATE_BASE_DIR")
	if baseDir == "" {
		tmp := os.Getenv("RENOVATE_TMPDIR")
		if tmp == "" {
			tmp = os.TempDir()
		}
		baseDir = filepath.Join(tmp, "renovate")
	}
	return &Resolver{
		BaseDir:  baseDir,
		Platform: os.Getenv("RENOVATE_PLATFORM"),
	}
}

// Watch must be called before renovate starts, since renovate deletes the repo clone,
// and its config.json, once it finishes with the repo.
func (r *Resolver) Watch(ctx context.Context, repo string) *Watch {
	slug, _, _ := strings.Cut(repo, "?") // repo may carry options, eg "?loglevel=debug"
	path := filepath.Join(r.BaseDir, "repos", r.Platform, filepath.FromSlash(slug), "config.json")

	interval := r.PollInterval
	if interval <= 0 {
		interval = defaultPollInterval
	}

	ctx, cancel := context.WithCancel(ctx)
	w := &Watch{cancel: cancel, done: make(chan struct{}), team: Unknown, name: slug}

	go func() {
		defer close(w.done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			if team, name, found := read(path, slug); found {
				w.team, w.name = team, name // safe: written before done closes
				return
			}
			select {
			case <-ctx.Done():
				logrus.Debugf("repoowner: no config.json seen for %s, defaulting team=%q", slug, Unknown)
				return
			case <-ticker.C:
			}
		}
	}()

	return w
}

type Watch struct {
	cancel context.CancelFunc
	done   chan struct{}
	team   string
	name   string
}

// Result stops the watch, falling back to team "unknown" and name = repo slug.
func (w *Watch) Result() (team, name string) {
	w.cancel()
	<-w.done
	return w.team, w.name
}

// read sets found once the file exists, so a broken config stops the watch.
func read(path, slug string) (team, name string, found bool) {
	data, err := os.ReadFile(path) // #nosec G304 -- path built from trusted env + discovered repo slug
	if err != nil {
		return "", "", false
	}

	team, name = Unknown, slug

	var config repoConfig
	if err := json.Unmarshal(data, &config); err != nil {
		logrus.Warnf("repoowner: cannot parse config.json in %s, defaulting team=%q: %s", slug, Unknown, err)
		return team, name, true
	}

	if config.Team != "" {
		team = config.Team
	}
	if config.ServiceName != "" {
		name = config.ServiceName
	}
	return team, name, true
}
