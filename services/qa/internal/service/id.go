package service

import (
	"fmt"
	"sync/atomic"
	"time"
)

var idCounter uint64

func newID(prefix string) string {
	next := atomic.AddUint64(&idCounter, 1)
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UTC().UnixNano(), next)
}
