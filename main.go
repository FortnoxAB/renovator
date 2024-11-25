//go:generate go install github.com/vektra/mockery/v2@v2.45.1
//go:generate mockery
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fortnoxab/renovator/pkg/agent"
	"github.com/fortnoxab/renovator/pkg/master"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)
	defer stop()
	err := app().RunContext(ctx, os.Args)
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
}

func app() *cli.App {
	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = "renovator"
	app.Usage = "Split renovate workload across multiple agents using redis"
	app.Before = globalBefore

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:  "loglevel",
			Value: "info",
			Usage: "available levels are: " + strings.Join(getLevels(), ","),
		},
	}

	redisStringflag := &cli.StringFlag{
		Name:  "redis-url",
		Usage: "redis url, ex redis://[[username]:[password]]@localhost:6379/0 or rediss://[[username]:[password]]@localhost:6379/0 for tls/ssl connections",
	}

	app.Commands = []*cli.Command{
		{
			Name:  "master",
			Usage: "run renovate master",
			Action: func(ctx *cli.Context) error {
				m, err := master.NewMasterFromContext(ctx)
				if err != nil {
					return err
				}
				return m.Run(ctx.Context)
			},
			Flags: []cli.Flag{
				redisStringflag,
				&cli.BoolFlag{
					Name:  "leaderelect",
					Usage: "run leader election",
					Value: false,
				},
				&cli.DurationFlag{
					Name:  "election-ttl",
					Usage: "leader election ttl",
					Value: 10 * time.Second,
				},
				&cli.StringFlag{
					Name:  "schedule",
					Usage: "Run discovery on a schedule instead of onetime, value is a standard cron string",
				},
				&cli.BoolFlag{
					Name:  "run-first-time",
					Usage: "run discovery directly, only applicable if a schedule is provided",
				},
			},
		},
		{
			Name:  "agent",
			Usage: "run renovate agent",
			Action: func(ctx *cli.Context) error {
				a, err := agent.NewAgentFromContext(ctx)
				if err != nil {
					return err
				}
				a.Run(ctx.Context)
				return nil
			},
			Flags: []cli.Flag{
				redisStringflag,
				&cli.IntFlag{
					Name:  "max-process-count",
					Value: 1,
					Usage: "Defines the maximum amount of simultaneous renovate processes",
				},
			},
		},
	}

	return app
}

func globalBefore(c *cli.Context) error {
	lvl, err := logrus.ParseLevel(c.String("loglevel"))
	if err != nil {
		return err
	}
	if lvl != logrus.InfoLevel {
		_, _ = fmt.Fprintf(os.Stderr, "using loglevel: %s\n", lvl.String())
	}
	logrus.SetLevel(lvl)

	return nil
}

func getLevels() []string {
	lvls := make([]string, len(logrus.AllLevels))
	for k, v := range logrus.AllLevels {
		lvls[k] = v.String()
	}
	return lvls
}
