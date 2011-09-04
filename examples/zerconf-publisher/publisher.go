package main

import (
	"net"
	
	dns "github.com/miekg/godns"
	"github.com/davecheney/zeroconf"
)

var (
	zone      = zeroconf.NewLocalZone()
)

func main() {
	zone.Add <- &zeroconf.Entry {
		Publish: true,
		RR: &dns.RR_A {
			Hdr: dns.RR_Header {
				Name: "stora.local.",
				Class: dns.ClassINET,
				Ttl: 5,
			},
			A: net.IPv4(192,168,1,200),	
		},
	}
				
	c := make(chan bool)
	<- c
}
