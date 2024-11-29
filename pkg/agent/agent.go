package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fortnoxab/renovator/pkg/command"
	localredis "github.com/fortnoxab/renovator/pkg/redis"
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
	ZombieReaper()
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
					start := time.Now()
					err := a.Renovator.RunRenovate(repo)
					if err != nil {
						renovateRuns.WithLabelValues("error", repo).Inc()
						logrus.Errorf("error renovating repo: %s err: %s", repo, err)
						continue
					}
					renovateRuns.WithLabelValues("ok", repo).Inc()
					logrus.Infof("finished renovating repo: %s in %s", repo, time.Since(start))
				}
			}
		}()
	}

	for ctx.Err() == nil {
		repos, err := a.RedisClient.BLPop(ctx, time.Second*5, localredis.RedisRepoListKey).Result() // 0 duration == block until key exists. We block for 5 seconds otherwise it will not work with shutdown https://github.com/redis/go-redis/issues/2556
		if err != nil {
			if err == redis.Nil {
				continue
			}
			logrus.Error("BLpop err: ", err)
			continue
		}

		if len(repos) != 2 || repos[0] != localredis.RedisRepoListKey {
			logrus.Errorf("unexpected reply from BLpop: %s", strings.Join(repos, ","))
			continue
		}

		reposToProcess <- repos[1]
	}
	wg.Wait()
}

func ZombieReaper() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGCHLD)

	go func() {
		for range signals {
			for {
				var wstatus syscall.WaitStatus
				time.Sleep(1 * time.Second)
				pid, err := syscall.Wait4(-1, &wstatus, 0, nil)
				if errors.Is(err, syscall.ECHILD) {
					break
				}
				if err != nil {
					logrus.Errorf("error waiting for child %d: %s", pid, err)
					continue
				}
				logrus.Debugf("reaped zombie %d %d", pid, wstatus)
			}
		}
	}()
}
