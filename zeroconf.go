package zeroconf

// convenience routines

import (
	"net"
	
	dns "github.com/miekg/godns"
)

func PublishA(z *Zone, name string, ip net.IP) {
        z.Add <- &Entry{
                Publish: true,
                RR: &dns.RR_A{
                        Hdr: dns.RR_Header{
                                Name:   name,
                                Ttl:    60,
                                Class:  dns.ClassINET,
                                Rrtype: dns.TypeA,
                        },
                        A: ip,
                },
        }
}

func PublishPTR(z *Zone, name, target string) {
        z.Add <- &Entry{
                Publish: true,
                RR: &dns.RR_PTR{
                        Hdr: dns.RR_Header{
                                Name:   name,
                                Ttl:    60,
                                Class:  dns.ClassINET,
                                Rrtype: dns.TypePTR,
                        },
                        Ptr: target,
                },
        }
}

func PublishSRV(z *Zone, name, target string, port uint16) {
	z.Add <- &Entry {
                Publish: true,
                RR: &dns.RR_SRV{
                        Hdr: dns.RR_Header{
                                Name:   name,
                                Ttl:    60,
                                Class:  dns.ClassINET,
                                Rrtype: dns.TypeSRV,
                        },
                        Priority: 10,
                        Weight:   10,
                        Port:     port,
                        Target:   target,
                },
        }
}
