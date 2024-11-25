package leaderelect

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

const lock = "lock.renovator-leader"

type Candidate struct {
	id          string
	redisClient redis.Cmdable
	sessionTTL  time.Duration
}

func NewCandidate(rc redis.Cmdable, electionDur time.Duration) *Candidate {
	id := fmt.Sprintf("candidate-%d", rand.Int64())
	return &Candidate{
		id:          id,
		redisClient: rc,
		sessionTTL:  electionDur,
	}
}

func (c *Candidate) IsLeader(ctx context.Context) (bool, error) {
	isLeader, err := c.redisClient.SetNX(ctx, lock, c.id, c.sessionTTL).Result()

	if err == nil && isLeader {
		logrus.Debug("aquired new leader lock")
		return true, nil
	}

	leaderId, err := c.redisClient.Get(ctx, lock).Result()
	if err != nil {
		return false, err
	}
	logrus.Debugf("current leader is: %s", leaderId)
	return leaderId == c.id, nil
}
