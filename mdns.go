// Advertise network services via multicast DNS
package mdns

import (
	"errors"
	"log"
	"net"

	dns "github.com/miekg/godns"
)

var (
	ipv4mcastaddr = &net.UDPAddr{
		IP:   net.ParseIP("224.0.0.251"),
		Port: 5353,
	}

	ipv6mcastaddr = &net.UDPAddr{
		IP:   net.ParseIP("ff02::fb"),
		Port: 5353,
	}
	local *zone // the local mdns zone
)

func init() {
	local = &zone{
		entries: make(map[string]entries),
		add:     make(chan *entry, 16),
		queries: make(chan *query, 16),
	}
	go local.mainloop()
	if err := local.listen(ipv4mcastaddr); err != nil {
		log.Fatalf("Failed to listen %s: %s", ipv4mcastaddr, err)
	}
	if err := local.listen(ipv6mcastaddr); err != nil {
		log.Printf("Failed to listen %s: %s", ipv6mcastaddr, err)
	}
}

// Add an A record. fqdn should be fully qualified, including local.
func PublishA(fqdn string, ttl int, ip net.IP) {
	local.add <- &entry{
		RR: &dns.RR_A{
			Hdr: dns.RR_Header{
				Name:   fqdn,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    uint32(ttl),
			},
			A: ip,
		},
	}
}

// Publish a PTR record, fqdn should be fully qualified, including local.
func PublishPTR(fqdn string, ttl int, target string) {
	local.add <- &entry{
		RR: &dns.RR_PTR{
			Hdr: dns.RR_Header{
				Name:   fqdn,
				Rrtype: dns.TypePTR,
				Class:  dns.ClassINET,
				Ttl:    uint32(ttl),
			},
			Ptr: target,
		},
	}
}

// Publish a SRV record, fqdn and target should be fully qualified, including local.
func PublishSRV(fqdn string, ttl int, target string, port int) {
	local.add <- &entry{
		RR: &dns.RR_SRV{
			Hdr: dns.RR_Header{
				Name:   fqdn,
				Rrtype: dns.TypeSRV,
				Class:  dns.ClassINET,
				Ttl:    uint32(ttl),
			},
			Target: target,
			Port:   uint16(port),
		},
	}
}

// Publish a TXT record, fqdn should be fully qualified, including local.
func PublishTXT(fqdn string, ttl int, txt string) {
	local.add <- &entry{
		RR: &dns.RR_TXT{
			Hdr: dns.RR_Header{
				Name:   fqdn,
				Rrtype: dns.TypeTXT,
				Class:  dns.ClassINET,
				Ttl:    uint32(ttl),
			},
			Txt: txt,
		},
	}
}

type entry struct {
	dns.RR
}

func (e *entry) fqdn() string {
	return e.Header().Name
}

type query struct {
	dns.Question
	result chan *entry
}

type entries []*entry

func (e entries) contains(entry *entry) bool {
	for _, ee := range e {
		if equals(entry, ee) {
			return true
		}
	}
	return false
}

type zone struct {
	entries map[string]entries
	add     chan *entry // add entries to zone
	queries chan *query // query exsting entries in zone
}

func (z *zone) mainloop() {
	for {
		select {
		case entry := <-z.add:
			if !z.entries[entry.fqdn()].contains(entry) {
				z.entries[entry.fqdn()] = append(z.entries[entry.fqdn()], entry)
			}
		case q := <-z.queries:
			for _, entry := range z.entries[q.Question.Name] {
				if q.matches(entry) {
					q.result <- entry
				}
			}
			close(q.result)
		}
	}
}

func (z *zone) query(q dns.Question) (entries []*entry) {
	res := make(chan *entry, 16)
	z.queries <- &query{q, res}
	for e := range res {
		entries = append(entries, e)
	}
	return
}

func (q *query) matches(entry *entry) bool {
	return q.Question.Qtype == dns.TypeANY || q.Question.Qtype == entry.RR.Header().Rrtype
}

func equals(this, that *entry) bool {
	if _, ok := this.RR.(*dns.RR_ANY); ok {
		return true // *RR_ANY matches anything
	}
	if _, ok := that.RR.(*dns.RR_ANY); ok {
		return true // *RR_ANY matches all
	}
	return false
}

type connector struct {
	*net.UDPAddr
	*net.UDPConn
	*zone
}

func (z *zone) listen(addr *net.UDPAddr) error {
	conn, err := openSocket(addr)
	if err != nil {
		return err
	}
	if err := conn.JoinGroup(nil, addr.IP); err != nil {
		return err
	}
	c := &connector{
		UDPAddr: addr,
		UDPConn: conn,
		zone:    z,
	}
	go c.mainloop()
	return nil
}

func openSocket(addr *net.UDPAddr) (*net.UDPConn, error) {
	switch addr.IP.To4() {
	case nil:
		return net.ListenUDP("udp6", &net.UDPAddr{
			IP:   net.IPv6zero,
			Port: addr.Port,
		})
	default:
		return net.ListenUDP("udp4", &net.UDPAddr{
			IP:   net.IPv4zero,
			Port: addr.Port,
		})
	}
	panic("unreachable")
}

func (c *connector) mainloop() {
	in := make(chan struct {
		*dns.Msg
		*net.UDPAddr
	}, 32)
	go func() {
		for {
			msg, addr, err := c.readMessage()
			if err != nil {
				// log dud packets
				log.Printf("Cound not read from %s: %s", c.UDPConn, err)
			}
			if msg.IsQuestion() {
				in <- struct {
					*dns.Msg
					*net.UDPAddr
				}{msg, addr}
			}
		}
	}()
	for {
		msg := <-in
		msg.MsgHdr.Response = true // convert question to response
		for _, result := range c.query(msg.Question) {
			msg.Answer = append(msg.Answer, result.RR)
		}
		msg.Extra = append(msg.Extra, c.findExtra(msg.Answer...)...)
		if len(msg.Answer) > 0 {
			if err := c.writeMessage(msg.Msg); err != nil {
				log.Fatalf("Cannot send: %s", err)
			}
		}
	}
}

func (c *connector) query(qs []dns.Question) (results []*entry) {
	for _, q := range qs {
		results = append(results, c.zone.query(q)...)
	}
	return
}

// recursively probe for related records
func (c *connector) findExtra(r ...dns.RR) (extra []dns.RR) {
	for _, rr := range r {
		var q dns.Question
		switch rr := rr.(type) {
		case *dns.RR_PTR:
			q = dns.Question{
				Name:   rr.Ptr,
				Qtype:  dns.TypeANY,
				Qclass: dns.ClassINET,
			}
		case *dns.RR_SRV:
			q = dns.Question{
				Name:   rr.Target,
				Qtype:  dns.TypeA,
				Qclass: dns.ClassINET,
			}
		default:
			continue
		}
		res := c.zone.query(q)
		if len(res) > 0 {
			for _, entry := range res {
				extra = append(append(extra, entry.RR), c.findExtra(entry.RR)...)
			}
		}
	}
	return
}

// encode an mdns msg and broadcast it on the wire
func (c *connector) writeMessage(msg *dns.Msg) (err error) {
	if buf, ok := msg.Pack(); ok {
		_, err = c.WriteToUDP(buf, c.UDPAddr)
	}
	return
}

// consume an mdns packet from the wire and decode it
func (c *connector) readMessage() (*dns.Msg, *net.UDPAddr, error) {
	buf := make([]byte, 1500)
	read, addr, err := c.ReadFromUDP(buf)
	if err != nil {
		return nil, nil, err
	}
	if msg := new(dns.Msg); msg.Unpack(buf[:read]) {
		return msg, addr, nil
	}
	return nil, addr, errors.New("Unable to unpack buffer")
}
