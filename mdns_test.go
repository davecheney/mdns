package zeroconf

import (
	"testing"
	"time"
)

var (
	LOCAL = NewLocalZone()
)

func TestPublish(t *testing.T) {
	<-time.After(60e9)
}
