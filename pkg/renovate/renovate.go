package renovate

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fortnoxab/renovator/pkg/command"
)

type Runner struct {
	commander command.Commander
}

func NewRunner(c command.Commander) *Runner {
	return &Runner{
		commander: c,
	}
}

func (r *Runner) RunRenovate(repo string) error {
	_, _, _, err := r.commander.Run("renovate", repo)
	if err != nil {
		return fmt.Errorf("error running renovate on repo: %s, err: %w", repo, err)
	}
	return nil
}

// DoAutoDiscover returns a list of repos
func (r *Runner) DoAutoDiscover() ([]string, error) {

	file, err := createTempFile()
	if err != nil {
		return nil, fmt.Errorf("error creating tempfile, err: %w", err)
	}
	defer os.Remove(file.Name())

	stdOut, stdErr, _, err := r.commander.Run("renovate", "--write-discovered-repos", file.Name())
	if err != nil {
		return nil, fmt.Errorf("error running renovate discovery, err: %w, stdOut: '%s', stdErr: '%s'", err, stdOut, stdErr)
	}

	fileData, err := os.ReadFile(file.Name())
	if err != nil {
		return nil, fmt.Errorf("error reading repolist file, err: %w", err)
	}

	repos := []string{}

	err = json.Unmarshal(fileData, &repos)
	if err != nil {
		return nil, fmt.Errorf("error unmarshling repolist, err: %w", err)
	}

	return repos, nil

}

func createTempFile() (*os.File, error) {
	file, err := os.CreateTemp("", "renovator_")
	if err != nil {
		return nil, err
	}

	err = file.Close()
	if err != nil {
		return nil, err
	}
	return file, nil
}
