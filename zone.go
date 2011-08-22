package zeroconf

import (
	"log"

	dns "github.com/miekg/godns"
)

var (
	Local = &Zone{
		Domain:    "local.",
		entries:   make(map[string]entries),
		additions: make(chan *Entry),
		questions: make(chan *query),
	}
)

func init() {
	go Local.mainloop()
}

type Entry struct {
	expires int  // the timestamp when this record will expire
	publish bool // whether this entry should be broadcast in response to an mDNS question
	rr      dns.RR
}

func (e *Entry) fqdn() string {
	return e.rr.Header().Name
}

type query struct {
	question dns.Question
	response chan *Entry
}

type entries []*Entry

type Zone struct {
	Domain    string
	entries   map[string]entries
	additions chan *Entry
	questions chan *query
}

func (z *Zone) Add(entry *Entry) {
	log.Printf("Add: %#v", entry)
	z.additions <- entry
}

func (z *Zone) Query(q dns.Question) <-chan *Entry {
	query := &query{q, make(chan *Entry)}
	z.questions <- query
	return query.response
}

func (z *Zone) mainloop() {
	for {
		select {
		case addition := <-z.additions:
			z.add(addition)
		case q := <-z.questions:
			z.query(q)
		}
	}
}

func (z *Zone) add(entry *Entry) {
	z.entries[entry.fqdn()] = append(z.entries[entry.fqdn()], entry)
	log.Printf("%#v", entry)
}

func (z *Zone) query(query *query) {

}

func Publish(rr dns.RR) {
	Local.Add(&Entry{
		expires: 2 ^ 31, // never
		publish: true,
		rr:      rr,
	})
}
