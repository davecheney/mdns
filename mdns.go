package mdns

import (
	"log"
	"net"
	"os"
	"strings"
	"time"

	dns "github.com/miekg/godns"
)

const seconds = 1e9

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
		entries:   make(map[string]entries),
		add:       make(chan *entry, 16),
		queries:   make(chan *query, 16),
		subscribe: make(chan *query, 16),
	}
	go local.mainloop()
	if err := local.listen(ipv4mcastaddr); err != nil {
		log.Fatal("Failed to listen: ", err)
	}
	if err := local.listen(ipv6mcastaddr); err != nil {
		log.Fatal("Failed to listen: ", err)
	}
}

// Add an A record. fqdn should be the full domain name, including .local.
func PublishA(fqdn string, ttl int, ip net.IP) {
	local.add <- &entry{
		publish: true,
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

func PublishPTR(fqdn string, ttl int, target string) {
	local.add <- &entry{
		publish: true,
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

func PublishSRV(fqdn string, ttl int, target string, port int) {
	local.add <- &entry{
		publish: true,
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

func PublishTXT(fqdn string, ttl int, txt string) {
	local.add <- &entry{
		publish: true,
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

type Entry interface {
	Domain() string
	Name() string
	Type() string
}

type entry struct {
	Expires int64 // the timestamp when this record will expire in nanoseconds
	publish bool  // whether this entry should be broadcast in response to an mDNS question
	dns.RR
	Source *net.UDPAddr
}

func (e *entry) fqdn() string {
	return e.Header().Name
}

func (e *entry) Domain() string {
	return "local." // TODO
}

func (e *entry) Name() string {
	return strings.Split(e.fqdn(), ".")[0]
}

func (e *entry) Type() string {
	return e.fqdn()[len(e.Name()+".") : len(e.fqdn())-len(e.Domain())]
}

func (e *entry) equals(entry Entry) bool {
	return true
}

type query struct {
	dns.Question
	result chan Entry
}

type entries []*entry

func (e entries) contains(entry *entry) bool {
	for _, ee := range e {
		if equals(entry.RR, ee.RR) {
			return true
		}
	}
	return false
}

type zone struct {
	entries       map[string]entries
	add           chan *entry // add entries to zone
	queries       chan *query // query exsting entries in zone
	subscribe     chan *query // subscribe to new entries added to zone
	subscriptions []*query
}

func (z *zone) mainloop() {
	for {
		select {
		case entry := <-z.add:
			if !z.entries[entry.fqdn()].contains(entry) {
				z.entries[entry.fqdn()] = append(z.entries[entry.fqdn()], entry)
				z.publish(entry)
			}
		case q := <-z.queries:
			for _, entry := range z.entries[q.Question.Name] {
				if q.matches(entry) {
					q.result <- entry
				}
			}
			close(q.result)
		case q := <-z.subscribe:
			z.subscriptions = append(z.subscriptions, q)
		}
	}
}

func Subscribe(t uint16) chan Entry {
	res := make(chan Entry, 16)
	local.subscribe <- &query{
		dns.Question{
			"",
			t,
			dns.ClassINET,
		},
		res,
	}
	return res
}

func (z *zone) query(q dns.Question) (entries []*entry) {
	res := make(chan Entry, 16)
	z.queries <- &query{q, res}
	for e := range res {
		entries = append(entries, e.(*entry))
	}
	return
}

func (z *zone) publish(entry *entry) {
	for _, c := range z.subscriptions {
		if c.matches(entry) {
			c.result <- entry
		}
	}
}

func (q *query) matches(entry *entry) bool {
	return q.Question.Qtype == dns.TypeANY || q.Question.Qtype == entry.RR.Header().Rrtype
}

func equals(this, that dns.RR) bool {
	if _, ok := this.(*dns.RR_ANY); ok {
		return true // *RR_ANY matches anything
	}
	if _, ok := that.(*dns.RR_ANY); ok {
		return true // *RR_ANY matches all
	}
	return false
}

type connector struct {
	*net.UDPAddr
	*net.UDPConn
	*zone
}

func (z *zone) listen(addr *net.UDPAddr) os.Error {
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

func openSocket(addr *net.UDPAddr) (*net.UDPConn, os.Error) {
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
				log.Fatalf("Cound not read from %s: %s", c.UDPConn, err)
			}
			in <- struct {
				*dns.Msg
				*net.UDPAddr
			}{msg, addr}
		}
	}()
	for {
		select {
		case msg := <-in:
			if msg.IsQuestion() {
				r := new(dns.Msg)
				r.MsgHdr.Response = true
				r.Question = msg.Question
				for _, result := range c.query(msg.Question) {
					if result.publish {
						r.Answer = append(r.Answer, result.RR)
					}
				}
				r.Extra = append(r.Extra, c.findExtra(r.Answer...)...)
				log.Printf("%s", r)
				if len(r.Answer) > 0 {
					if err := c.writeMessage(r); err != nil {
						log.Fatalf("Cannot send: %s", err)
					}

				}
			} else {
				for _, rr := range msg.Answer {
					c.add <- &entry{
						Expires: time.Nanoseconds() + int64(rr.Header().Ttl*seconds),
						publish: false,
						RR:      rr,
						Source:  msg.UDPAddr,
					}
				}
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
				if entry.publish {
					extra = append(append(extra, entry.RR), c.findExtra(entry.RR)...)
				}
			}
		}
	}
	return
}

// encode an mdns msg and broadcast it on the wire
func (c *connector) writeMessage(msg *dns.Msg) (err os.Error) {
	if buf, ok := msg.Pack(); ok {
		_, err = c.WriteToUDP(buf, c.UDPAddr)
	}
	return
}

// consume an mdns packet from the wire and decode it
func (c *connector) readMessage() (*dns.Msg, *net.UDPAddr, os.Error) {
	buf := make([]byte, 1500)
	read, addr, err := c.ReadFromUDP(buf)
	if err != nil {
		return nil, nil, err
	}
	if msg := new(dns.Msg); msg.Unpack(buf[:read]) {
		return msg, addr, nil
	}
	return nil, addr, os.NewError("Unable to unpack buffer")
}
