package main

import (
	"net"

	"github.com/davecheney/mdns"
)

func main() {
	mdns.PublishA("stora.local.", 3600, net.IPv4(192, 168, 1, 200))
	mdns.PublishPTR("_ssh._tcp.local.", 3600, "stora._ssh._tcp.local.")
	mdns.PublishSRV("stora._ssh._tcp.local.", 3600, "stora.local.", 22)
	mdns.PublishTXT("stora._ssh._tcp.local.", 3600, "")

	mdns.PublishA("router.local.", 3600, net.IPv4(192, 168, 1, 254))
	mdns.PublishPTR("_ssh._tcp.local.", 3600, "router._ssh._tcp.local.")
	mdns.PublishSRV("router._ssh._tcp.local.", 3600, "router.local.", 22)
	mdns.PublishTXT("router._ssh._tcp.local.", 3600, "")

	<-make(chan bool)
}
