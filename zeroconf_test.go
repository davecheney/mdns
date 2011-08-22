package zeroconf

import (
	"testing"
	"time"
	
	dns "github.com/miekg/godns"
)

func TestPublish(t *testing.T) {
	service := dns.NewRR(dns.TypeSRV).(*dns.RR_SRV)
	service.Header().Name = "host._http._tcp.local."
	service.Priority = 10
	service.Weight = 10
	service.Port = 80
	service.Target = "lucky.local"
	Publish(service)
	
	<-time.After(60e9)
}
