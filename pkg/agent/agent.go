package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/fortnoxab/renovator/pkg/command"
	"github.com/fortnoxab/renovator/pkg/master"
	"github.com/fortnoxab/renovator/pkg/renovate"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

type Agent struct {
	Renovator       *renovate.Runner
	RedisClient     redis.Cmdable
	MaxProcessCount int
}

func NewAgentFromContext(cCtx *cli.Context) (*Agent, error) {
	opt, err := redis.ParseURL(cCtx.String("redis-url"))
	if err != nil {
		return nil, fmt.Errorf("error parsing redis url, err: %w", err)
	}
	rc := redis.NewClient(opt)
	return &Agent{
		Renovator:       renovate.NewRunner(&command.Exec{}),
		RedisClient:     rc,
		MaxProcessCount: cCtx.Int("max-process-count"),
	}, nil
}

func (a *Agent) Run(ctx context.Context) {

	reposToProcess := make(chan string)

	wg := &sync.WaitGroup{}

	for i := 0; i < a.MaxProcessCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {

				select {
				case <-ctx.Done():
					return
				case repo := <-reposToProcess:
					logrus.Infof("running renovate on repo: %s", repo)
					err := a.Renovator.RunRenovate(repo)
					if err != nil {
						logrus.Errorf("error renovating repo: %s err: %s", repo, err)
						continue
					}
					logrus.Infof("finished renovating repo: %s", repo)
				}

			}
		}()
	}

	for {
		if ctx.Err() != nil {
			break
		}
		repos, err := a.RedisClient.BLPop(ctx, 0, master.RedisRepoListKey).Result() // 0 duration == block until key exists.
		if err != nil {
			logrus.Error("BLpop err: ", err)
			continue
		}

		logrus.Debugf("got %d number of repos to process", len(repos))

		if len(repos) != 2 || repos[0] != master.RedisRepoListKey {
			logrus.Errorf("unexpected reply from BLpop: %s", strings.Join(repos, ","))
			continue
		}

		reposToProcess <- repos[1]
	}
	wg.Wait()
}
