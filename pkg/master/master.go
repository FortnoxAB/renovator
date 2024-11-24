package master

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fortnoxab/renovator/pkg/command"
	"github.com/fortnoxab/renovator/pkg/leaderelect"
	"github.com/fortnoxab/renovator/pkg/renovate"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

const RedisRepoListKey = "renovator-joblist"

const electionTTL = 2 * time.Minute

type Master struct {
	Renovator    *renovate.Runner
	RedisClient  redis.Cmdable
	LeaderElect  bool
	CronSchedule cron.Schedule
	RunFirstTime bool
}

type autoDiscoverJob struct {
	ctx            context.Context
	redisClient    redis.Cmdable
	renovateRunner *renovate.Runner
	doLeaderElect  bool
}

func NewMasterFromContext(cCtx *cli.Context) (*Master, error) {
	opt, err := redis.ParseURL(cCtx.String("redis-url"))
	if err != nil {
		return nil, fmt.Errorf("error parsing redis url, err: %w", err)
	}
	rc := redis.NewClient(opt)

	var cronSchedule cron.Schedule
	if cs := cCtx.String("schedule"); cs != "" {
		cronSchedule, err = cron.ParseStandard(cs)
		if err != nil {
			return nil, fmt.Errorf("failed to create cron schedule, err: %w", err)
		}
	}
	return &Master{
		Renovator:    renovate.NewRunner(&command.Exec{}),
		RedisClient:  rc,
		LeaderElect:  cCtx.Bool("leaderelect"),
		CronSchedule: cronSchedule,
		RunFirstTime: cCtx.Bool("run-first-time"),
	}, nil
}

func (m *Master) Run(ctx context.Context) error {

	if m.CronSchedule == nil {
		return doRun(ctx, m.RedisClient, m.Renovator, m.LeaderElect)
	}

	if m.RunFirstTime {
		err := doRun(ctx, m.RedisClient, m.Renovator, m.LeaderElect)
		if err != nil {
			return err
		}
	}

	cronRnr := cron.New()

	job := autoDiscoverJob{
		redisClient:    m.RedisClient,
		renovateRunner: m.Renovator,
		doLeaderElect:  m.LeaderElect,
	}

	cronRnr.Schedule(m.CronSchedule, job)

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		logrus.Debug("running cron")
		cronRnr.Run()
		wg.Done()
	}()

	// If context is cancelled, stop cronrunner and wait for job to finish
	<-ctx.Done()
	logrus.Debug("main context cancelled")

	logrus.Debug("waiting for cronrunner to stop...")
	<-cronRnr.Stop().Done()
	logrus.Debug("cronrunner stopped")

	wg.Wait()

	return nil
}

func doRun(ctx context.Context, redisClient redis.Cmdable, renovateRunner *renovate.Runner, doLeaderElect bool) error {

	if doLeaderElect {
		candidate := leaderelect.NewCandidate(redisClient, electionTTL)

		isLeader, err := candidate.Elect(ctx)
		if err != nil {
			return fmt.Errorf("failed to elect leader, err: %w", err)
		}

		if !isLeader {
			logrus.Debug("lost election, exiting")
			return nil
		}
		logrus.Debug("won election, running repo discovery")
	}

	logrus.Debug("running renovate autodiscover")
	repos, err := renovateRunner.DoAutoDiscover()
	if err != nil {
		return err
	}

	logrus.Debug("pushing repo list to redis")
	err = redisClient.RPush(ctx, RedisRepoListKey, repos).Err()
	if err != nil {
		return fmt.Errorf("failed to push repolist to redis, err: %w", err)
	}
	return nil
}

func (j autoDiscoverJob) Run() {
	logrus.Debug("running autodiscovery")
	err := doRun(j.ctx, j.redisClient, j.renovateRunner, j.doLeaderElect)
	if err != nil {
		logrus.Errorf("error when running autodiscovery, err: %s", err.Error())
	}
	logrus.Debug("completed autodiscovery")
}
