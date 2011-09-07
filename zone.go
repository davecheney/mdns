package zeroconf

import (
	"log"
	"net"
	"strings"

	dns "github.com/miekg/godns"
)

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
