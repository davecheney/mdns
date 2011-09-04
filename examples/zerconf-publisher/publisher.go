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
	zone.Add <- &zeroconf.Entry{
		Publish: true,
		RR: &dns.RR_A{
			Hdr: dns.RR_Header{
				Name:   "stora.local.",
				Ttl:    5,
				Class:  dns.ClassINET,
				Rrtype: dns.TypeA,
			},
			A: net.IPv4(192, 168, 1, 200),
		},
	}
	zone.Add <- &zeroconf.Entry {
		Publish: true,
		RR: &dns.RR_PTR {
			Hdr: dns.RR_Header{
				Name: 	"_ssh._tcp.local.",
				Ttl:	5,
				Class:	dns.ClassINET,
				Rrtype:	dns.TypePTR,
			},
			Ptr:	"stora._ssh._tcp.local.",
		},
	}
        zone.Add <- &zeroconf.Entry {
                Publish: true,
                RR: &dns.RR_SRV {
                        Hdr: dns.RR_Header{
                                Name:   "stora._ssh._tcp.local.",
                                Ttl:    5,
                                Class:  dns.ClassINET,
                                Rrtype: dns.TypeSRV,
                        },
			Priority: 10,
			Weight: 10,
			Port:	22,
                        Target:    "stora.local.",
                },
        }

	<- make(chan bool)
}
