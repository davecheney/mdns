package zeroconf

import (
	"testing"
	"time"

	dns "github.com/miekg/godns"
)

var (
	LOCAL = NewLocalZone()
)

func TestPublish(t *testing.T) {
	m := new(dns.Msg)
	m.SetQuestion("_ssh._tcp.local.", dns.TypeANY)
	LOCAL.listener.writeMessage(m)

	<-time.After(60e9)
}
