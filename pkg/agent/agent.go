package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/fortnoxab/renovator/pkg/command"
	"github.com/fortnoxab/renovator/pkg/master"
	"github.com/fortnoxab/renovator/pkg/renovate"
	"github.com/fortnoxab/renovator/pkg/webserver"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var renovateRuns = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "renovate_runs",
	Help: "Number of renovate runs",
}, []string{"result", "repo"})

func init() {
	prometheus.MustRegister(renovateRuns)
}

type Agent struct {
	Renovator       *renovate.Runner
	RedisClient     redis.Cmdable
	MaxProcessCount int
	Webserver       *webserver.Webserver
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
		Webserver:       &webserver.Webserver{Port: cCtx.String("port"), EnableMetrics: true},
	}, nil
}

func (a *Agent) Run(ctx context.Context) {

	reposToProcess := make(chan string)

	wg := &sync.WaitGroup{}
	if a.Webserver != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.Webserver.Start(ctx)
		}()
	}

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
						renovateRuns.WithLabelValues("error", repo).Inc()
						logrus.Errorf("error renovating repo: %s err: %s", repo, err)
						continue
					}
					renovateRuns.WithLabelValues("ok", repo).Inc()
					logrus.Infof("finished renovating repo: %s", repo)
				}

			}
		}()
	}

	for ctx.Err() == nil {
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
