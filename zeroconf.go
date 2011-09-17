package zeroconf

// convenience routines

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	dns "github.com/miekg/godns"
)

// commented out till dns.NewRRString() works
//func Publish(z *Zone, rr string) {
//	val, err := dns.NewRRString(rr) 
//	if err != nil {
//		panic(err.String())
//	}
//	PublishRR(z, val)
//}

func PublishRR(z Zone, rr dns.RR) {
	z.Add(&Entry{
		Publish: true,
		RR:      rr,
	})
}

type Entry struct {
	Expires int64 // the timestamp when this record will expire in nanoseconds
	Publish bool  // whether this entry should be broadcast in response to an mDNS question
	RR      dns.RR
	Source  *net.UDPAddr
}

func (e *Entry) fqdn() string {
	return e.RR.Header().Name
}

func (e *Entry) Domain() string {
	return "local." // TODO
}

func (e *Entry) Name() string {
	return strings.Split(e.fqdn(), ".")[0]
}

func (e *Entry) Type() string {
	return e.fqdn()[len(e.Name()+".") : len(e.fqdn())-len(e.Domain())]
}

type Query struct {
	Question dns.Question
	Result   chan *Entry
}

type entries []*Entry

func (e entries) contains(entry *Entry) bool {
	for _, ee := range e {
		if equals(ee.RR, entry.RR) {
			return true
		}
	}
	return false
}

type zone struct {
	Domain        string
	entries       map[string]entries
	add           chan *Entry   // add entries to zone
	query         chan *Query   // query exsting entries in zone
	subscribe     chan *Query   // subscribe to new entries added to zone
	broadcast     chan *dns.Msg // send messages to listeners
	subscriptions []*Query
}

type Zone interface {
	Query(dns.Question) chan *Entry
	Subscribe(uint16) chan *Entry
	Add(*Entry)
}

func NewLocalZone() Zone {
	add, query, broadcast := make(chan *Entry, 16), make(chan *Query, 16), make(chan *dns.Msg, 16)
	z := &zone{
		Domain:    "local.",
		entries:   make(map[string]entries),
		add:       add,
		query:     query,
		broadcast: broadcast,
		subscribe: make(chan *Query, 16),
	}
	go z.mainloop()
	if err := z.listen(IPv4MCASTADDR); err != nil {
		log.Fatal("Failed to listen: ", err)
	}
	//        if err := listen(IPv6MCASTADDR, add, query, broadcast); err != nil {
	//               log.Fatal("Failed to listen: ", err)
	//        }
	return z
}

func (z *zone) mainloop() {
	for {
		select {
		case entry := <-z.add:
			z.add0(entry)
		case q := <-z.query:
			z.query0(q)
		case q := <-z.subscribe:
			z.subscriptions = append(z.subscriptions, q)
		}
	}
}

func (z *zone) Add(e *Entry) {
	z.add <- e
}

func (z *zone) Subscribe(t uint16) chan *Entry {
	res := make(chan *Entry, 16)
	z.subscribe <- &Query{
		dns.Question {
			"",
			dns.ClassINET,
			t,
		},
		res,
	}
	return res
}

func (z *zone) Query(q dns.Question) chan *Entry {
	res := make(chan *Entry, 16)
	z.query <- &Query{ q, res }
	return res
}

func (z *zone) add0(entry *Entry) {
	if !z.entries[entry.fqdn()].contains(entry) {
		z.entries[entry.fqdn()] = append(z.entries[entry.fqdn()], entry)
		z.publish(entry)
	}
}

func (z *zone) publish(entry *Entry) {
	for _, c := range z.subscriptions {
		if c.matches(entry) {
			c.Result <- entry
		} 
	}
}

func (z *zone) query0(query *Query) {
	for _, entry := range z.entries[query.Question.Name] {
		if query.matches(entry) { 
			query.Result <- entry
		}
	}
	close(query.Result)
}

func (q *Query) matches(entry *Entry) bool {
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

const (
	seconds = 1e9
)

var (
	IPv4MCASTADDR = &net.UDPAddr{
		IP:   net.ParseIP("224.0.0.251"),
		Port: 5353,
	}

	IPv6MCASTADDR = &net.UDPAddr{
		IP:   net.ParseIP("ff02::fb"),
		Port: 5353,
	}
)

type listener struct {
	addr    *net.UDPAddr
	*net.UDPConn
	Zone
}

func (z *zone) listen(addr *net.UDPAddr) os.Error {
	conn, err := openSocket(addr)
	if err != nil {
		return err
	}
	if err := conn.JoinGroup(nil, addr.IP); err != nil {
		return err
	}
	l := &listener{
		addr:    addr,
		UDPConn:    conn,
		Zone: z,
	}
	go l.mainloop()
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

func (l *listener) mainloop() {
	type incoming struct {
		*dns.Msg
		*net.UDPAddr
	}
	in := make(chan incoming, 32)
	out := make (chan *dns.Msg, 32)
	go func() {
		for {
			msg, addr, err := l.readMessage()
			if err != nil {
				log.Fatalf("Cound not read from %s: %s", l.UDPConn, err)
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
				for _, result := range l.query(msg.Question) {
					if result.Publish {
						r.Answer = append(r.Answer, result.RR)
					}
				}
				if len(r.Answer) > 0 {
					fmt.Println(msg.Msg)	
					fmt.Println(r)
					out <- r
				}
			} else {
				for _, rr := range msg.Answer {
					l.Add(&Entry{
						Expires: time.Nanoseconds() + int64(rr.Header().Ttl*seconds),
						Publish: false,
						RR:      rr,
						Source:  msg.UDPAddr,
					})
				}
			}
		case msg := <-out:
			if err := l.writeMessage(msg); err != nil {
				log.Fatalf("Cannot send: %s", err)
			}
		}
	}
}

func (l *listener) query(qs []dns.Question) []*Entry {
	result := make([]*Entry,0 )
	for _, q := range qs {
		for r := range l.Query(q) {
			result = append(result, r)
		}
	}
	return result
}

func (l *listener) writeMessage(msg *dns.Msg) (err os.Error) {
	if buf, ok := msg.Pack(); ok {
		_, err = l.WriteToUDP(buf, l.addr)
	}
	return
}

func (l *listener) readMessage() (*dns.Msg, *net.UDPAddr, os.Error) {
	buf := make([]byte, 1500)
	read, addr, err := l.ReadFromUDP(buf)
	if err != nil {
		return nil, nil, err
	}
	if msg := new(dns.Msg); msg.Unpack(buf[:read]) {
		return msg, addr, nil
	}
	return nil, addr, os.NewError("Unable to unpack buffer")
}
