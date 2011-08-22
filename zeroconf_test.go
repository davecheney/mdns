package zeroconf

import (
	"testing"
	"time"
	
	dns "github.com/miekg/godns"
)

func TestPublish(t *testing.T) {
	host := dns.NewRR(dns.TypeA)
	host.Header().Name = "host.local."
	Publish(host)

	service := dns.NewRR(dns.TypeSRV)
	service.Header().Name = "host._http._tcp.local."
	Publish(service)
	
	<-time.After(60e9)
}
