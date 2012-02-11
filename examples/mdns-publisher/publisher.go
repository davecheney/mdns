package main

import (
	"log"
	"github.com/davecheney/mdns"
)

func mustPublish(rr string){
	if err := mdns.Publish(rr) ; err != nil {
		log.Fatalf(`Unable to publish record "%s": %v`, rr, err)
	}
}

func main() {
	// A simple example. Publish an A record for my router at 192.168.1.254.
	mustPublish("router.local. 60 IN A 192.168.1.254")

	// A more compilcated example. Publish a SVR record for ssh running on port
	// 22 for my home NAS.

	// Publish an A record as before
	mustPublish("stora.local. 60 IN A 192.168.1.200")

	// Publish a PTR record for the _ssh._tcp DNS-SD type
	mustPublish("_ssh._tcp.local. 60 IN PTR stora._ssh._tcp.local.")

	// Publish a SRV record tying the _ssh._tcp record to an A record and a port.
	mustPublish("stora._ssh._tcp.local. 60 IN SRV 0 0 22 stora.local.")

	// Most mDNS browsing tools expect a TXT record for the service even if there
	// are not records defined by RFC 2782.
	mustPublish(`stora._ssh._tcp.local. 60 IN TXT ""`)

	select {}
}
