package kafka

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/IBM/sarama"
	localredis "github.com/fortnoxab/renovator/pkg/redis"
	"github.com/jonaz/mgit/pkg/bitbucket"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

func Start(ctx context.Context, brokers string, redisClient redis.Cmdable) {
	config := sarama.NewConfig()
	config.Version = sarama.V3_5_1_0
	group := "renovator-master"

	consumer := Consumer{redis: redisClient}
	client, err := sarama.NewConsumerGroup(strings.Split(brokers, ","), group, config)
	if err != nil {
		logrus.Errorf("Error creating consumer group client: %v", err)
		return
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			if err := client.Consume(ctx, []string{"vcs-pullrequests"}, &consumer); err != nil {
				// if errors.Is(err, sarama.ErrClosedConsumerGroup) {
				// 	return
				// }
				logrus.Errorf("Error from consumer: %v", err)
			}
			// check if context was cancelled, signaling that the consumer should stop
			if ctx.Err() != nil {
				return
			}
		}
	}()

	wg.Wait()
	if err = client.Close(); err != nil {
		logrus.Errorf("Error closing client: %v", err)
	}
}

// Consumer represents a Sarama consumer group consumer
type Consumer struct {
	redis redis.Cmdable
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (consumer *Consumer) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (consumer *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

type hookData struct {
	HookData bitbucket.WebhookEvent `json:"hookData"`
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
// Once the Messages() channel is closed, the Handler must finish its processing
// loop and exit.
func (consumer *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// NOTE:
	// Do not move the code below to a goroutine.
	// The `ConsumeClaim` itself is called within a goroutine, see:
	// https://github.com/IBM/sarama/blob/main/consumer_group.go#L27-L29
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				logrus.Info("message channel was closed")
				return nil
			}
			hookData := &hookData{}
			err := json.Unmarshal(message.Value, hookData)
			if err != nil {
				logrus.Errorf("failed to unmarshal event to struct: %s message was: %s", err, string(message.Value))
			}
			session.MarkMessage(message, "")

			hook := hookData.HookData
			if strings.HasPrefix(hook.PullRequest.Title, "rebase!") && hook.PullRequest.Title != hook.PreviousTitle {
				repo := hook.PullRequest.ToRef.Repository.Project.Key + "/" + hook.PullRequest.ToRef.Repository.Slug

				// If its a webhook and its already in the queue to be processed we move it first in the queue.
				err = consumer.redis.LRem(session.Context(), localredis.RedisRepoListKey, 0, repo).Err()
				if err != nil {
					logrus.Errorf("error LRem: %s", err)
					continue
				}

				logrus.Infof("trigger renovate on %s due to 'rebase!' in PR %s", repo, hook.PullRequest.Links.Self[0].Href)
				err = consumer.redis.LPush(session.Context(), localredis.RedisRepoListKey, []string{repo}).Err()
				if err != nil {
					logrus.Errorf("error LPush: %s", err)
				}
			}
		//TODO dbouncer same hook can happen twice....

		// Should return when `session.Context()` is done.
		// If not, will raise `ErrRebalanceInProgress` or `read tcp <ip>:<port>: i/o timeout` when kafka rebalance. see:
		// https://github.com/IBM/sarama/issues/1192
		case <-session.Context().Done():
			return nil
		}
	}
}
