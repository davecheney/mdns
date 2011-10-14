package mdns

import (
	"fmt"
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
                Domain:    "local.",
                entries:   make(map[string]entries),
                add:       make(chan *entry, 16),
                queries:     make(chan *Query, 16),
                subscribe: make(chan *Query, 16),
        }
        go local.mainloop()
        if err := local.listen(ipv4mcastaddr); err != nil {
                log.Fatal("Failed to listen: ", err)
        }
        if err := local.listen(ipv6mcastaddr); err != nil {
                log.Fatal("Failed to listen: ", err)
        }
}

type host struct {
	Name   string
	Domain string
	Addrs  []net.IP
}

type Service struct {
	host
	_type
	Port uint16
}

type proto int

func (p proto) String() string {
	if p == tcp {
		return "_tcp"
	}
	return "_udp"
}

const (
	tcp proto = iota
	udp
)

type _type struct {
	name string
	proto
}

var (
	Ssh = &_type{"_ssh", tcp}
)

func (s *Service) fqdn() string {
	return fmt.Sprintf("%s.%s", s.Name, s.Domain)
}

func (s *Service) service() string {
	return fmt.Sprintf("%s.%s.%s", s.name, s.proto.String(), s.Domain)
}

func (s *Service) serviceFqdn() string {
	return s.Name + "." + s.service()
}

func Publish(s *Service) {
	for _, addr := range s.Addrs {
		a := dns.NewRR(dns.TypeA).(*dns.RR_A)
		a.Hdr.Name = s.fqdn()
		a.Hdr.Class = dns.ClassINET
		a.Hdr.Ttl = 3600
		a.A = addr
		PublishRR(a)
	}

	ptr := dns.NewRR(dns.TypePTR).(*dns.RR_PTR)
	ptr.Hdr.Name = s.service()
	ptr.Hdr.Class = dns.ClassINET
	ptr.Hdr.Ttl = 3600
	ptr.Ptr = s.serviceFqdn()
	PublishRR(ptr)

	srv := dns.NewRR(dns.TypeSRV).(*dns.RR_SRV)
	srv.Hdr.Name = s.serviceFqdn()
	srv.Hdr.Class = dns.ClassINET
	srv.Hdr.Ttl = 3600
	srv.Port = s.Port
	srv.Target = s.fqdn()
	PublishRR(srv)

	txt := dns.NewRR(dns.TypeTXT).(*dns.RR_TXT)
	txt.Hdr.Name = s.serviceFqdn()
	txt.Hdr.Class = dns.ClassINET
	txt.Hdr.Ttl = 3600
	PublishRR(txt)
}

func PublishRR(rr dns.RR) {
	local.add <- &entry{
		Publish: true,
		RR:      rr,
	}
}

type Entry interface {
	Domain() string
	Name() string
	Type() string
	Header() *dns.RR_Header
}

type entry struct {
	Expires int64 // the timestamp when this record will expire in nanoseconds
	Publish bool  // whether this entry should be broadcast in response to an mDNS question
	dns.RR
	Source  *net.UDPAddr
}

func (e *entry) fqdn() string {
	return e.RR.Header().Name
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

type Query struct {
	dns.Question
	result   chan Entry
}

type entries []*entry

func (e entries) contains(entry Entry) bool {
	for _, ee := range e {
		if ee.equals(entry) {
			return true
		}
	}
	return false
}

type zone struct {
	Domain        string
	entries       map[string]entries
	add           chan *entry // add entries to zone
	queries         chan *Query // query exsting entries in zone
	subscribe     chan *Query // subscribe to new entries added to zone
	subscriptions []*Query
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
	local.subscribe <- &Query{
		dns.Question{
			"",
			t,
			dns.ClassINET,
		},
		res,
	}
	return res
}

func (z *zone) query(q dns.Question) (entries []*entry, extra []*entry) {
	res := make(chan Entry, 16)
	z.queries <- &Query{q, res}
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

func (q *Query) matches(entry Entry) bool {
	return q.Question.Qtype == dns.TypeANY || q.Question.Qtype == entry.Header().Rrtype
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
		zone: z,
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
	type incoming struct {
		*dns.Msg
		*net.UDPAddr
	}
	in := make(chan incoming, 32)
	go func() {
		for {
			msg, addr, err := c.readMessage()
			if err != nil {
				log.Fatalf("Cound not read from %s: %s", c.UDPConn, err)
			}
			in <- incoming{msg, addr}
		}
	}()

	for {
		select {
		case msg := <-in:
			if msg.IsQuestion() {
				r := new(dns.Msg)
				r.MsgHdr.Response = true
				results, additionals := c.query(msg.Question)
				for _, result := range results {
					if result.Publish {
						r.Answer = append(r.Answer, result.RR)
					}
				}
				for _, additional := range additionals {
					if additional.Publish {
						r.Extra = append(r.Extra, additional.RR)
					}
				}
				if len(r.Answer) > 0 {
					r.Extra = c.findAdditional(r.Answer)
					fmt.Println(r)
					if err := c.writeMessage(r); err != nil {
						log.Fatalf("Cannot send: %s", err)
					}

				}
			} else {
				for _, rr := range msg.Answer {
					c.add <- &entry{
						Expires: time.Nanoseconds() + int64(rr.Header().Ttl*seconds),
						Publish: false,
						RR:      rr,
						Source:  msg.UDPAddr,
					}
				}
			}
		}
	}
}

func (c *connector) findAdditional(rr []dns.RR) []dns.RR {
	return []dns.RR{}
}

func (c *connector) query(qs []dns.Question) (results []*entry, additionals []*entry) {
	for _, q := range qs {
		result, additional := c.zone.query(q)
		results = append(results, result...)
		additionals = append(additionals, additional...)
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
