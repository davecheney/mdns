package zeroconf

import (
	"testing"
	"time"

	dns "github.com/miekg/godns"
)

func TestPublish(t *testing.T) {
	m := new(dns.Msg)
	m.SetQuestion("_afpovertcp._tcp.local.", dns.TypeANY)
	Listener.writeMessage(m)

	<-time.After(60e9)
}
