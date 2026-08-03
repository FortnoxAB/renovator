package repoowner

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

const Unknown = "unknown"

// repoConfig is the subset of a repo's root config.json used for alert routing
type repoConfig struct {
	ServiceName string `json:"serviceName"`
	Team        string `json:"team"`
}

// Resolve reads the owning team and service name from a repo clone's config.json.
func Resolve(cloneDir, slug string) (team, name string) {
	team, name = Unknown, slug

	path := filepath.Join(cloneDir, "config.json")
	data, err := os.ReadFile(path) // #nosec G304 -- path built from trusted env + discovered repo slug
	if err != nil {
		// Plenty of repos have no config.json, so this is expected rather than a problem.
		logrus.Debugf("repoowner: cannot read %s, defaulting team=%q: %s", path, Unknown, err)
		return team, name
	}

	var config repoConfig
	if err := json.Unmarshal(data, &config); err != nil {
		logrus.Warnf("repoowner: cannot parse %s, defaulting team=%q: %s", path, Unknown, err)
		return team, name
	}

	if config.Team != "" {
		team = config.Team
	}
	if config.ServiceName != "" {
		name = config.ServiceName
	}
	return team, name
}
