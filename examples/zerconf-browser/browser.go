package main

import (
	"fmt"
	"time"

	dns "github.com/miekg/godns"
	"github.com/davecheney/zeroconf"
)

var (
	zone      = zeroconf.NewLocalZone()
	questions = []dns.Question{
		{"_ssh._tcp.local.", dns.TypeANY, dns.ClassINET},
	}
)

func main() {
	results := make(chan *zeroconf.Entry, 16)
	zone.Subscribe <- &zeroconf.Query{
		Question: dns.Question{
			"", // fake
			dns.TypeANY,
			dns.ClassINET,
		},
		Result: results,
	}
	for {
		select {
		case <-time.After(2e9):
			for _, q := range questions {
				msg := new(dns.Msg)
				msg.SetQuestion(q.Name, q.Qtype)
				// zone.Broadcast <- msg
			}
		case result := <-results:
			fmt.Printf("+ %-32s%32s %s\n", result.Name(), result.Type(), result.Domain())
		}
	}
}
