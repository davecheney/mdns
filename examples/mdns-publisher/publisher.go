package main

import (
	"net"

	"github.com/davecheney/mdns"
)

var (
	zone = mdns.NewLocalZone()
)

func main() {
	host := &mdns.Host{
		"stora",
		"local.",
		[]net.IP{net.IPv4(192, 168, 1, 200)},
	}
	service := &mdns.Service{
		host,
		mdns.Ssh,
		22,
	}

	mdns.Publish(zone, service)

	mdns.Publish(zone, &mdns.Service{
		&mdns.Host{
			"router",
			"local.",
			[]net.IP{net.IPv4(192, 168, 1, 254)},
		},
		mdns.Ssh,
		22,
	})

	<-make(chan bool)
}
