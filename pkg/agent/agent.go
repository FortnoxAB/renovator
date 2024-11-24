package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

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

func (a *Agent) Run(ctx context.Context) error {

	guard := make(chan struct{}, a.MaxProcessCount)

	wg := &sync.WaitGroup{}

outer:
	for {

		select {
		case <-ctx.Done():
			break outer
		case guard <- struct{}{}: // will block if guard channel is already filled

			repo, err := a.RedisClient.LPop(ctx, master.RedisRepoListKey).Result()
			if err != nil {

				if err == redis.Nil { // No values in redis queue, sleep to conserve resources
					logrus.Debugf("found no values in redis queue, sleeping")
					time.Sleep(100 * time.Millisecond) // TODO: Make this duration configurable?
				}

				// TODO: Also sleep if actual error???
				if err != redis.Nil {
					logrus.Errorf("error when popping value from redis list, err: %s", err.Error())
				}
				<-guard
				continue
			}

			logrus.Infof("running renovate on repo: %s", repo)

			wg.Add(1)
			go func() {

				//time.Sleep(2 * time.Second)

				err := a.Renovator.RunRenovate(repo)
				if err != nil {
					logrus.Error(err)
				}

				logrus.Infof("finished renovating repo: %s", repo)

				<-guard
				wg.Done()
			}()
		default:
			// TODO: Do we need this??
			// Sleep to conserve resources
			time.Sleep(100 * time.Millisecond)
		}

	}
	wg.Wait()

	return nil
}
