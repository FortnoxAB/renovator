package master

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
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
	m := &Master{
		Renovator:   renovate.NewRunner(commanderMock),
		RedisClient: redisMock,
		LeaderElect: false,
	}

	repoList := []string{"project1/repo1", "project1/repo2", "project2/repo1"}

	commanderMock.On("Run", "renovate", "--write-discovered-repos", mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			filePath := args[2].(string)
			assert.Regexp(t, regexp.MustCompile("^/tmp/renovator_\\d+$"), filePath)

			file, err := os.OpenFile(filePath, os.O_RDWR|os.O_TRUNC, 0600)
			assert.NoError(t, err)

			d, err := json.Marshal(repoList)
			assert.NoError(t, err)
			_, err = file.WriteAt(d, 0)
			assert.NoError(t, err)
		}).
		Return("", "", 0, nil).
		Once()

	redisMock.On("RPush", mock.Anything, "renovator-joblist", repoList).
		Return(redis.NewIntResult(3, nil)).
		Once()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	err := m.Run(ctx)
	assert.NoError(t, err)
}

func TestRunWithLeaderElect(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.RFC3339Nano, FullTimestamp: true})

	commanderMock := mocks.NewMockCommander(t)
	redisMock := mocks.NewMockCmdable(t)
	m := &Master{
		Renovator:   renovate.NewRunner(commanderMock),
		RedisClient: redisMock,
		LeaderElect: true,
	}

	redisMock.On("SetNX", mock.Anything, "lock.renovator-leader", mock.AnythingOfType("string"), 2*time.Minute).
		Return(redis.NewBoolResult(true, nil)).
		Once()

	repoList := []string{"project1/repo1", "project1/repo2", "project2/repo1"}

	commanderMock.On("Run", "renovate", "--write-discovered-repos", mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			filePath := args[2].(string)
			assert.Regexp(t, regexp.MustCompile("^/tmp/renovator_\\d+$"), filePath)

			file, err := os.OpenFile(filePath, os.O_RDWR|os.O_TRUNC, 0600)
			assert.NoError(t, err)

			d, err := json.Marshal(repoList)
			assert.NoError(t, err)
			_, err = file.WriteAt(d, 0)
			assert.NoError(t, err)
		}).
		Return("", "", 0, nil).
		Once()

	redisMock.On("RPush", mock.Anything, "renovator-joblist", repoList).
		Return(redis.NewIntResult(3, nil)).
		Once()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	err := m.Run(ctx)
	assert.NoError(t, err)
}

func TestRunWithLeaderElectAndLoosing(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.RFC3339Nano, FullTimestamp: true})

	commanderMock := mocks.NewMockCommander(t)
	redisMock := mocks.NewMockCmdable(t)
	m := &Master{
		Renovator:   renovate.NewRunner(commanderMock),
		RedisClient: redisMock,
		LeaderElect: true,
	}

	redisMock.On("SetNX", mock.Anything, "lock.renovator-leader", mock.AnythingOfType("string"), 2*time.Minute).
		Return(redis.NewBoolResult(false, nil)).
		Once()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	err := m.Run(ctx)
	assert.NoError(t, err)
}

func TestRunWithSchedule(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.RFC3339Nano, FullTimestamp: true})

	commanderMock := mocks.NewMockCommander(t)
	redisMock := mocks.NewMockCmdable(t)
	m := &Master{
		Renovator:    renovate.NewRunner(commanderMock),
		RedisClient:  redisMock,
		CronSchedule: NewTestCronSchedule(50*time.Millisecond, 3),
	}

	repoList := []string{"project1/repo1", "project1/repo2", "project2/repo1"}

	commanderMock.On("Run", "renovate", "--write-discovered-repos", mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			filePath := args[2].(string)
			assert.Regexp(t, regexp.MustCompile("^/tmp/renovator_\\d+$"), filePath)

			file, err := os.OpenFile(filePath, os.O_RDWR|os.O_TRUNC, 0600)
			assert.NoError(t, err)

			d, err := json.Marshal(repoList)
			assert.NoError(t, err)
			_, err = file.WriteAt(d, 0)
			assert.NoError(t, err)
		}).
		Return("", "", 0, nil).
		Times(3)

	redisMock.On("RPush", mock.Anything, "renovator-joblist", repoList).
		Return(redis.NewIntResult(3, nil)).
		Times(3)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		runErr := m.Run(ctx)
		assert.NoError(t, runErr)
		wg.Done()
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()
	wg.Wait()
}

func TestRunWithScheduleAndLeaderElection(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.RFC3339Nano, FullTimestamp: true})

	commanderMock := mocks.NewMockCommander(t)
	redisMock := mocks.NewMockCmdable(t)
	m := Master{
		Renovator:    renovate.NewRunner(commanderMock),
		RedisClient:  redisMock,
		CronSchedule: NewTestCronSchedule(50*time.Millisecond, 3),
		LeaderElect:  true,
	}

	redisMock.On("SetNX", mock.Anything, "lock.renovator-leader", mock.AnythingOfType("string"), 2*time.Minute).
		Return(redis.NewBoolResult(true, nil)).
		Times(3)

	repoList := []string{"project1/repo1", "project1/repo2", "project2/repo1"}

	commanderMock.On("Run", "renovate", "--write-discovered-repos", mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			filePath := args[2].(string)
			assert.Regexp(t, regexp.MustCompile("^/tmp/renovator_\\d+$"), filePath)

			file, err := os.OpenFile(filePath, os.O_RDWR|os.O_TRUNC, 0600)
			assert.NoError(t, err)

			d, err := json.Marshal(repoList)
			assert.NoError(t, err)
			_, err = file.WriteAt(d, 0)
			assert.NoError(t, err)
		}).
		Return("", "", 0, nil).
		Times(3)

	redisMock.On("RPush", mock.Anything, "renovator-joblist", repoList).
		Return(redis.NewIntResult(3, nil)).
		Times(3)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		runErr := m.Run(ctx)
		assert.NoError(t, runErr)
		wg.Done()
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()
	wg.Wait()
}

type testCronSchedule struct {
	times []time.Time
}

func NewTestCronSchedule(interval time.Duration, runs int) testCronSchedule {

	times := []time.Time{}
	last := time.Now()
	for i := 0; i < runs; i++ {
		x := last.Add(interval)
		times = append(times, x)
		last = x
	}
	return testCronSchedule{times: times}
}

func (s testCronSchedule) Next(t time.Time) time.Time {
	for _, x := range s.times {
		if x.After(t) {
			return x
		}
	}
	return time.Time{}
}
