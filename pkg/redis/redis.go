package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const RedisRepoListKey = "renovator-joblist"

func RemoveAlreadyQueued(ctx context.Context, redisClient redis.Cmdable, repos []string) ([]string, error) {
	var reposToQueue []string
	reposInQueue, err := redisClient.LRange(ctx, RedisRepoListKey, 0, -1).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("error from LRange: %w", err)
	}

	for _, repo := range repos {
		add := true
		for _, e := range reposInQueue {
			if repo == e {
				add = false
				break
			}
		}
		if add {
			reposToQueue = append(reposToQueue, repo)
		}
	}
	return reposToQueue, nil
}
