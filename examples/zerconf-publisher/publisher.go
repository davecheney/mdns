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
	a := dns.NewRR(dns.TypeA).(*dns.RR_A)
	a.Hdr.Name = "stora.local."
	a.Hdr.Class = dns.ClassINET
	a.Hdr.Ttl = 3600
	a.A = net.IPv4(192, 168, 1, 200)
	zeroconf.PublishRR(zone, a)

	ptr := dns.NewRR(dns.TypePTR).(*dns.RR_PTR)
	ptr.Hdr.Name = "_ssh._tcp.local."
	ptr.Hdr.Class = dns.ClassINET
	ptr.Hdr.Ttl = 3600
	ptr.Ptr = "stora._ssh._tcp.local."
	zeroconf.PublishRR(zone, ptr)

	srv := dns.NewRR(dns.TypeSRV).(*dns.RR_SRV)
	srv.Hdr.Name = "stora._ssh._tcp.local."
	srv.Hdr.Class = dns.ClassINET
	srv.Hdr.Ttl = 3600
	srv.Port = 22
	srv.Target = "stora.local."
	zeroconf.PublishRR(zone, srv)

	txt := dns.NewRR(dns.TypeTXT).(*dns.RR_TXT)
	txt.Hdr.Name = "stora._ssh._tcp.local."
	txt.Hdr.Class = dns.ClassINET
	txt.Hdr.Ttl = 3600
	zeroconf.PublishRR(zone, txt)

	<-make(chan bool)
}
