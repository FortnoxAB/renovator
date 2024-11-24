package agent

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/fortnoxab/renovator/mocks"
	"github.com/fortnoxab/renovator/pkg/renovate"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRun(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.RFC3339Nano, FullTimestamp: true})

	commanderMock := mocks.NewMockCommander(t)
	redisMock := mocks.NewMockCmdable(t)
	a := &Agent{
		Renovator:       renovate.NewRunner(commanderMock),
		RedisClient:     redisMock,
		MaxProcessCount: 2,
	}

	redisMockList := redisMockList{
		list: []string{"project1/repo1", "project1/repo2", "project2/repo1"},
	}

	redisMockCall := redisMock.On("LPop", mock.Anything, "renovator-joblist")
	redisMockCall.RunFn = func(a mock.Arguments) {
		redisMockCall.ReturnArguments = mock.Arguments{redisMockList.LPop()}
	}

	commanderMock.On("Run", "renovate", "project1/repo1").
		Run(func(args mock.Arguments) {
			time.Sleep(200 * time.Millisecond)
		}).
		Return("", "", 0, nil).
		Once()
	commanderMock.On("Run", "renovate", "project1/repo2").
		Run(func(args mock.Arguments) {
			time.Sleep(200 * time.Millisecond)
		}).
		Return("", "", 0, nil).
		Once()
	commanderMock.On("Run", "renovate", "project2/repo1").
		Run(func(args mock.Arguments) {
			time.Sleep(200 * time.Millisecond)
		}).
		Return("", "", 0, nil).
		Once()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := a.Run(ctx)
	assert.NoError(t, err)
}

type redisMockList struct {
	lock sync.RWMutex
	list []string
}

func (t *redisMockList) LPop() *redis.StringCmd {
	t.lock.Lock()
	defer t.lock.Unlock()

	if len(t.list) == 0 {
		return redis.NewStringResult("", redis.Nil)
	}
	// Take first value and shift remaining
	first := t.list[0]
	t.list = t.list[1:]
	return redis.NewStringResult(first, nil)
}
