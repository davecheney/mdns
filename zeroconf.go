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

func PublishA(z *Zone, name string, ip net.IP) {
        z.Add <- &Entry{
                Publish: true,
                RR: &dns.RR_A{
                        Hdr: dns.RR_Header{
                                Name:   name,
                                Ttl:    60,
                                Class:  dns.ClassINET,
                                Rrtype: dns.TypeA,
                        },
                        A: ip,
                },
        }
}

func PublishPTR(z *Zone, name, target string) {
        z.Add <- &Entry{
                Publish: true,
                RR: &dns.RR_PTR{
                        Hdr: dns.RR_Header{
                                Name:   name,
                                Ttl:    60,
                                Class:  dns.ClassINET,
                                Rrtype: dns.TypePTR,
                        },
                        Ptr: target,
                },
        }
}

func PublishSRV(z *Zone, name, target string, port uint16) {
	z.Add <- &Entry {
                Publish: true,
                RR: &dns.RR_SRV{
                        Hdr: dns.RR_Header{
                                Name:   name,
                                Ttl:    60,
                                Class:  dns.ClassINET,
                                Rrtype: dns.TypeSRV,
                        },
                        Priority: 10,
                        Weight:   10,
                        Port:     port,
                        Target:   target,
                },
        }
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
	return e.fqdn()[len(e.Name()+"."):len(e.fqdn())-len(e.Domain())]
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

type Zone struct {
	Domain        string
	entries       map[string]entries
	Add           chan *Entry   // add entries to zone
	Query         chan *Query   // query exsting entries in zone
	Subscribe     chan *Query   // subscribe to new entries added to zone
	Broadcast     chan *dns.Msg // send messages to listeners
	subscriptions []*Query
}

func NewLocalZone() *Zone {
	add, query, broadcast := make(chan *Entry, 16), make(chan *Query, 16), make(chan *dns.Msg, 16)
	z := &Zone{
		Domain:    "local.",
		entries:   make(map[string]entries),
		Add:       add,
		Query:     query,
		Broadcast: broadcast,
		Subscribe: make(chan *Query, 16),
	}
	go z.mainloop()
	if err := listen(IPv4MCASTADDR, add, query, broadcast); err != nil {
		log.Fatal("Failed to listen: ", err)
	}
//        if err := listen(IPv6MCASTADDR, add, query, broadcast); err != nil {
//               log.Fatal("Failed to listen: ", err)
//        }
	return z
}

func (z *Zone) mainloop() {
	for {
		select {
		case entry := <-z.Add:
			z.add(entry)
		case q := <-z.Query:
			z.query(q)
		case q := <-z.Subscribe:
			z.subscriptions = append(z.subscriptions, q)
		}
	}
}

func (z *Zone) add(entry *Entry) {
	if !z.entries[entry.fqdn()].contains(entry) {
		z.entries[entry.fqdn()] = append(z.entries[entry.fqdn()], entry)
		z.publish(entry)
	}
}

func (z *Zone) publish(entry *Entry) {
	for _, c := range z.subscriptions {
		// TODO(dfc) use non blocking send
		c.Result <- entry
	}
}

func (z *Zone) query(query *Query) {
	for _, entry := range z.entries[query.Question.Name] {
		if query.Question.Qtype == entry.RR.Header().Rrtype {
			query.Result <- entry
		}
	}
	close(query.Result)
}

func equals(this, that dns.RR) bool {
	if _,ok := this.(*dns.RR_ANY) ; ok {
		return true // *RR_ANY matches anything
	}
	if _, ok := that.(*dns.RR_ANY) ; ok {
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
	conn    *net.UDPConn
	add     chan *Entry // send entries to zone
	query   chan *Query // send questions to zone
	publish chan *dns.Msg
}

func listen(addr *net.UDPAddr, add chan *Entry, query chan *Query, publish chan *dns.Msg) os.Error {
	conn, err := openSocket(addr)
	if err != nil {
		return err
	}
	if err := conn.JoinGroup(nil, addr.IP); err != nil {
		return err
	}
	l := &listener{
		addr:    addr,
		conn:    conn,
		add:     add,
		query:   query,
		publish: publish,
	}
	go l.mainloop()
	go l.publisher()
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
	for {
		msg, addr, err := l.readMessage()
		if err != nil {
			log.Fatalf("Cound not read from %s: %s", l.conn, err)
		}
		if msg.IsQuestion() {
			for _, question := range msg.Question {
				results := make(chan *Entry, 16)
				l.query <- &Query{question, results}
				for result := range results {
					if result.Publish {
						msg.Answer = append(msg.Answer, result.RR)
					}
				}
			}
			if len(msg.Answer) > 0 {
				msg.MsgHdr.Response = true
				fmt.Println(msg)
				l.writeMessage(msg)
				// l.publish <- msg
			}
		} else {
			for _, rr := range msg.Answer {
				l.add <- &Entry{
					Expires: time.Nanoseconds() + int64(rr.Header().Ttl*seconds),
					Publish: false,
					RR:      rr,
					Source:  addr,
				}
			}
		}
	}
}

func (l *listener) writeMessage(msg *dns.Msg) (err os.Error) {
	if buf, ok := msg.Pack(); ok {
		_, err = l.conn.WriteToUDP(buf, l.addr)
	}
	return
}

func (l *listener) publisher() {
	for msg := range l.publish {
		if err := l.writeMessage(msg); err != nil {
			log.Fatalf("Cannot send: %s", err)
		}
	}
	panic("publisher exited")
}

func (l *listener) readMessage() (*dns.Msg, *net.UDPAddr, os.Error) {
	buf := make([]byte, 1500)
	read, addr, err := l.conn.ReadFromUDP(buf)
	if err != nil {
		return nil, nil, err
	}
	if msg := new(dns.Msg); msg.Unpack(buf[:read]) {
		return msg, addr, nil
	}
	return nil, addr, os.NewError("Unable to unpack buffer")
}
