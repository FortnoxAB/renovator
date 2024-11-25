package leaderelect

import (
	"context"
	"testing"
	"time"

	"github.com/fortnoxab/renovator/mocks"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestElect(t *testing.T) {

	redisMock := mocks.NewMockCmdable(t)
	candidate := NewCandidate(redisMock, 200*time.Millisecond)

	redisMock.On("SetNX", mock.Anything, "lock.renovator-leader", candidate.id, candidate.sessionTTL).
		Return(redis.NewBoolResult(true, nil))

	isLeader, err := candidate.IsLeader(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, true, isLeader)
}

func TestElect2(t *testing.T) {

	redisMock := mocks.NewMockCmdable(t)
	candidate := NewCandidate(redisMock, 200*time.Millisecond)

	redisMock.On("SetNX", mock.Anything, "lock.renovator-leader", candidate.id, candidate.sessionTTL).
		Return(redis.NewBoolResult(false, nil))
	redisMock.On("Get", mock.Anything, "lock.renovator-leader").
		Return(redis.NewStringResult(candidate.id, nil))

	isLeader, err := candidate.IsLeader(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, true, isLeader)
}
func TestElect3(t *testing.T) {

	redisMock := mocks.NewMockCmdable(t)
	candidate := NewCandidate(redisMock, 200*time.Millisecond)

	redisMock.On("SetNX", mock.Anything, "lock.renovator-leader", candidate.id, candidate.sessionTTL).
		Return(redis.NewBoolResult(false, nil))
	redisMock.On("Get", mock.Anything, "lock.renovator-leader").
		Return(redis.NewStringResult("this id is not leader", nil))

	isLeader, err := candidate.IsLeader(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, false, isLeader)
}
