package main

import (
	"fmt"
	"time"

	dns "github.com/miekg/godns"
	"github.com/davecheney/zeroconf"
)

var (
	zone = zeroconf.NewLocalZone()
	questions = []dns.Question {
		{ "_ssh._tcp.local.", dns.TypeANY, dns.ClassINET },
	}
)

func printer() {
	results := make (chan *zeroconf.Entry, 16)
	zone.Subscribe <- &zeroconf.Query {
		Question: dns.Question {
			"", 
			dns.TypeDNSKEY, 
			dns.ClassINET,
		},
		Result: results,
	}
	for result := range results {	
		fmt.Printf("%#v", result)
	}	
}

func main() {
	go printer()

	for {
		<- time.After(2e9)
	}	
}
