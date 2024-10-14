package calculator

import (
	"github.com/google/uuid"
	"sync"
	"time"
)

var requestMutex sync.Mutex
var requestNumber uint64 = 0

var ServerId string

func init() {
	ServerId = uuid.New().String()
}

func GetRequestNumber() uint64 {
	requestMutex.Lock()
	defer requestMutex.Unlock()
	requestNumber++
	return requestNumber
}

func GetCurrentTimeInHumanReadableDate() (int64, string) {
	now := time.Now().UTC()
	now_ns := now.UnixNano()
	return now_ns, now.Format("2006-01-02T15:04:05.000Z")
}
