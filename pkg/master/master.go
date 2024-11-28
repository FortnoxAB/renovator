package master

import (
	"context"
	"fmt"
	"sync"

	"github.com/fortnoxab/renovator/pkg/agent"
	"github.com/fortnoxab/renovator/pkg/command"
	"github.com/fortnoxab/renovator/pkg/kafka"
	"github.com/fortnoxab/renovator/pkg/leaderelect"
	localredis "github.com/fortnoxab/renovator/pkg/redis"
	"github.com/fortnoxab/renovator/pkg/renovate"
	"github.com/fortnoxab/renovator/pkg/webserver"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

type Master struct {
	Renovator    *renovate.Runner
	RedisClient  redis.Cmdable
	Candidate    *leaderelect.Candidate
	LeaderElect  bool
	CronSchedule cron.Schedule
	RunFirstTime bool
	Webserver    *webserver.Webserver
	Brokers      string
}

type autoDiscoverJob struct {
	ctx            context.Context
	redisClient    redis.Cmdable
	renovateRunner *renovate.Runner
	doLeaderElect  bool
	candidate      *leaderelect.Candidate
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
			return nil, fmt.Errorf("failed to create cron schedule from value: '%s', err: %w", cs, err)
		}
	}
	return &Master{
		Renovator:    renovate.NewRunner(&command.Exec{}),
		Candidate:    leaderelect.NewCandidate(rc, cCtx.Duration("election-ttl")),
		RedisClient:  rc,
		LeaderElect:  cCtx.Bool("leaderelect"),
		CronSchedule: cronSchedule,
		RunFirstTime: cCtx.Bool("run-first-time"),
		Webserver:    &webserver.Webserver{Port: cCtx.String("port"), EnableMetrics: true},
		Brokers:      cCtx.String("kafka-brokers"),
	}, nil
}

func (m *Master) Run(ctx context.Context) error {
	if m.CronSchedule == nil {
		return doRun(ctx, m.Candidate, m.RedisClient, m.Renovator, m.LeaderElect)
	}
	agent.ZombieReaper()

	if m.RunFirstTime {
		logrus.Info("running due to --run-first-time")
		err := doRun(ctx, m.Candidate, m.RedisClient, m.Renovator, m.LeaderElect)
		if err != nil {
			logrus.Errorf("failed first time run, err: %s", err.Error())
		}
		logrus.Info("--run-first-time done")
	}

	cronRnr := cron.New()

	job := autoDiscoverJob{
		redisClient:    m.RedisClient,
		renovateRunner: m.Renovator,
		doLeaderElect:  m.LeaderElect,
		candidate:      m.Candidate,
		ctx:            ctx,
	}

	cronRnr.Schedule(m.CronSchedule, job)

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		logrus.Debug("running cron")
		cronRnr.Run()
	}()

	if m.Webserver != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.Webserver.Start(ctx)
		}()
	}

	if m.Brokers != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			kafka.Start(ctx, m.Brokers, m.RedisClient)
		}()
	}

	// If context is cancelled, stop cronrunner and wait for job to finish
	<-ctx.Done()
	logrus.Debug("main context cancelled")

	logrus.Debug("waiting for cronrunner to stop...")
	<-cronRnr.Stop().Done()
	logrus.Debug("cronrunner stopped")

	wg.Wait()

	return nil
}

func doRun(ctx context.Context, candidate *leaderelect.Candidate, redisClient redis.Cmdable, renovateRunner *renovate.Runner, doLeaderElect bool) error {

	if doLeaderElect {
		isLeader, err := candidate.IsLeader(ctx)
		if err != nil {
			return fmt.Errorf("failed to elect leader, err: %w", err)
		}

		if !isLeader {
			logrus.Debug("lost election, noop")
			return nil
		}
		logrus.Debug("won election, running repo discovery")
	}

	logrus.Debug("running renovate autodiscover")
	repos, err := renovateRunner.DoAutoDiscover()
	if err != nil {
		return err
	}

	reposToQueue, err := localredis.RemoveAlreadyQueued(ctx, redisClient, repos)
	if err != nil {
		return err
	}

	if len(reposToQueue) == 0 {
		logrus.Warn("zero repos to push to redis")
		return nil
	}

	logrus.Debug("pushing repo list to redis")
	err = redisClient.RPush(ctx, localredis.RedisRepoListKey, reposToQueue).Err()
	if err != nil {
		return fmt.Errorf("failed to push repolist to redis, err: %w", err)
	}
	return nil
}

func (j autoDiscoverJob) Run() {
	logrus.Debug("running autodiscovery")
	err := doRun(j.ctx, j.candidate, j.redisClient, j.renovateRunner, j.doLeaderElect)
	if err != nil {
		logrus.Errorf("error when running autodiscovery, err: %s", err.Error())
	}
	logrus.Debug("completed autodiscovery")
}
