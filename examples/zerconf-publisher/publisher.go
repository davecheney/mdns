package main

import (
	"net"

	dns "github.com/miekg/godns"
	"github.com/davecheney/zeroconf"
)

var (
	zone = zeroconf.NewLocalZone()
)

func main() {
	rr := &dns.RR_A{
		Hdr: dns.RR_Header{
			Name:   "stora.local.",
			Ttl:    5,
			Class:  dns.ClassINET,
			Rrtype: dns.TypeA,
		},
		A: net.IPv4(192, 168, 1, 200),
	}
	zone.Add <- &zeroconf.Entry{
		Publish: true,
		RR:      rr,
	}

	c := make(chan bool)
	<-c
}
