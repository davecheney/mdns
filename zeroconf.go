package zeroconf

import (
	dns "github.com/miekg/godns"
)

var (
	Local    = NewZone("local.")
	Listener = Listen(Local) // start mcast listner
)

func Publish(rr dns.RR) {
	Local.Add(&Entry{
		expires: 2 << 29, // never
		publish: true,
		rr:      rr,
	})
}
