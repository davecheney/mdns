package main

import (
	"fmt"

	dns "github.com/miekg/godns"
	"github.com/davecheney/mdns"
)

func main() {
	results := mdns.Subscribe(dns.TypeANY)
	for {
		result := <-results
		fmt.Printf("+ %-32s%32s %s\n", result.Name(), result.Type(), result.Domain())
	}
}
