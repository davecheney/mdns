package zeroconf

import (
	"log"

	dns "github.com/miekg/godns"
)

type Entry struct {
	Expires int64 // the timestamp when this record will expire in nanoseconds
	Publish bool  // whether this entry should be broadcast in response to an mDNS question
	RR      dns.RR
}

func (e *Entry) fqdn() string {
	return e.RR.Header().Name
}

type Query struct {
	Question dns.Question
	Result chan *Entry
}

type entries []*Entry

type Zone struct {
	Domain    string
	entries   map[string]entries
	Add chan *Entry
	Query chan *Query
	Subscribe chan *Query
	subscriptions []*Query
}

func NewLocalZone() *Zone {
	add, query := make(chan *Entry, 16), make(chan *Query, 16)
	z := &Zone{
		Domain:    "local.",
		entries:   make(map[string]entries),
		Add: add,
		Query: query,
		Subscribe: make(chan *Query, 16),
	}
	go z.mainloop()
	if err := listen(openSocket(IPv4MCASTADDR), add, query) ; err != nil {
		log.Fatal("Failed to listen: ", err)
	}
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
	z.entries[entry.fqdn()] = append(z.entries[entry.fqdn()], entry)
	log.Printf("Add: %s %#v", entry.fqdn(), entry)
}

func (z *Zone) query(query *Query) {
	for _, entry := range z.entries[query.Question.Name] {
		query.Result <- entry
	}
	close(query.Result)
	log.Printf("Query: %#v", query)
}

