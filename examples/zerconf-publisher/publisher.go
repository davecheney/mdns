package main

import (
	"net"

	"github.com/davecheney/zeroconf"
)

var (
	zone = zeroconf.NewLocalZone()
)

func main() {
	host := &zeroconf.Host{
		"stora",
		"local.",
		[]net.IP{net.IPv4(192, 168, 1, 200)},
	}
	service := &zeroconf.Service{
		host,
		zeroconf.Ssh,
		22,
	}

	zeroconf.Publish(zone, service)

	zeroconf.Publish(zone, &zeroconf.Service{
		&zeroconf.Host{
			"router",
			"local.",
			[]net.IP{net.IPv4(192, 168, 1, 254)},
		},
		zeroconf.Ssh,
		22,
	})

	<-make(chan bool)
}
