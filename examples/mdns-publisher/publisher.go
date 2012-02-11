package main

import (
	"github.com/davecheney/mdns"
)

func main() {
	// A simple example. Publish an A record for my router at 192.168.1.254.
	mdns.Publish("router.local. 3600 A 192, 168, 1, 254")

	// A more compilcated example. Publish a SVR record for ssh running on port
	// 22 for my home NAS.

	// Publish an A record as before
	mdns.Publish("stora.local. 3600 A 192, 168, 1, 200")

	// Publish a PTR record for the _ssh._tcp DNS-SD type
	mdns.Publish("_ssh._tcp.local. 3600 PTR stora._ssh._tcp.local.")

	// Publish a SRV record tying the _ssh._tcp record to an A record and a port.
	mdns.Publish("stora._ssh._tcp.local. 3600 SRV stora.local. 22")

	// Most mDNS browsing tools expect a TXT record for the service even if there
	// are not records defined by RFC 2782.
	mdns.Publish(`stora._ssh._tcp.local. 3600 ""`)

	select {}
}
