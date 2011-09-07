package main

import (
	"net"

	"github.com/davecheney/zeroconf"
)

var (
	zone = zeroconf.NewLocalZone()
)

func main() {
	zeroconf.PublishA(zone, "stora.local.", net.IPv4(192, 168, 1, 200))
	zeroconf.PublishPTR(zone, "_ssh._tcp.local.", "stora._ssh._tcp.local.")
	zeroconf.PublishSRV(zone, "stora._ssh._tcp.local.", "stora.local.", 22)

	<-make(chan bool)
}
