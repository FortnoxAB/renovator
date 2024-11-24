package leaderelect

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/redis/go-redis/v9"
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

func (c *Candidate) Elect(ctx context.Context) (isLeader bool, err error) {
	isLeader, err = c.redisClient.SetNX(ctx, lock, c.id, c.sessionTTL).Result()
	return
}

func (c *Candidate) IsLeader(ctx context.Context) (bool, error) {
	leaderId, err := c.redisClient.Get(ctx, lock).Result()
	if err != nil {
		return false, err
	}
	return leaderId == c.id, nil
}

// type ballot struct {
// 	identity   string
// 	expiration time.Time
// }

// func (c *Candidate) createBallot() ballot {
// 	return ballot{
// 		identity:   c.id,
// 		expiration: time.Now().Add(c.sessionTTL),
// 	}
// }

// func (c *Candidate) DoElection(ctx context.Context, sessionTTL time.Duration) (bool, error) {

// 	b := c.createBallot()

// 	isLeader, err := rc.SetNX(ctx, lock, b, sessionTTL).Result()

// 	return true, nil
// }
