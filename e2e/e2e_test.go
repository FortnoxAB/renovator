package e2e

// import (
// 	"context"
// 	"sync"
// 	"testing"
// 	"time"

// 	"github.com/fortnoxab/renovator/mocks"
// 	"github.com/fortnoxab/renovator/pkg/master"
// 	"github.com/fortnoxab/renovator/pkg/renovate"
// 	"github.com/redis/go-redis/v9"
// )

// func TestMasterLeaderLocking(t *testing.T) {
// 	commanderMock1 := mocks.NewMockCommander(t)
// 	redisMock1 := mocks.NewMockCmdable(t)
// 	m1 := &master.Master{
// 		Renovator:   renovate.NewRunner(commanderMock1),
// 		RedisClient: redisMock1,
// 		LeaderElect: true,
// 	}

// 	commanderMock2 := mocks.NewMockCommander(t)
// 	redisMock2 := mocks.NewMockCmdable(t)
// 	m2 := &master.Master{
// 		Renovator:   renovate.NewRunner(commanderMock2),
// 		RedisClient: redisMock2,
// 		LeaderElect: true,
// 	}

// 	wg := &sync.WaitGroup{}

// 	ctx1, cancel1 := context.WithTimeout(context.Background(), 100*time.Millisecond)
// 	defer cancel1()

// 	wg.Add(1)
// 	go func() {
// 		m1.Run(ctx1)
// 		wg.Done()
// 	}()

// 	ctx2, cancel2 := context.WithTimeout(context.Background(), 100*time.Millisecond)
// 	defer cancel2()

// 	wg.Add(1)
// 	go func() {
// 		m2.Run(ctx2)
// 		wg.Done()
// 	}()

// 	wg.Wait()
// }

// type redisMockKeyValue struct {
// 	lock sync.RWMutex
// 	data map[string][]interface{}
// }

// func (t *redisMockKeyValue) SetNX(key string, value []interface{}, expiration time.Duration) *redis.BoolCmd {
// 	t.lock.Lock()
// 	defer t.lock.Unlock()

// 	timeNow := time.Now()
// 	expTime := timeNow.Add(expiration)

// 	v, ok := t.data[key]
// 	if !ok {
// 		t.data[key] = []interface{}{value, expTime}
// 		return redis.NewBoolResult(true, nil)
// 	}

// 	//oldValue := v[0].(string)
// 	oldExpTime := v[1].(time.Time)

// 	if timeNow.After(oldExpTime) {
// 		t.data[key] = []interface{}{value, expTime}
// 		return redis.NewBoolResult(true, nil)
// 	}

// 	return redis.NewBoolResult(false, nil)
// }
