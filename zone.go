package zeroconf

import (
	"log"
	"os"

	dns "github.com/miekg/godns"
)

type Entry struct {
	expires int64 // the timestamp when this record will expire in nanoseconds
	publish bool  // whether this entry should be broadcast in response to an mDNS question
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
	listener  *listener
}

func NewLocalZone() *Zone {
	z := &Zone{
		Domain:    "local.",
		entries:   make(map[string]entries),
		additions: make(chan *Entry, 16),
		questions: make(chan *query, 16),
	}
	var err os.Error
	if z.listener, err = listen(z) ; err != nil {
		log.Fatal(err)
	}	
	go z.mainloop()
	go z.listener.mainloop()
	return z
}

func (z *Zone) Add(entry *Entry) {
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
	log.Printf("Add: %s %#v", entry.fqdn(), entry)
}

func (z *Zone) query(query *query) {
	for _, entry := range z.entries[query.question.Name] {
		query.response <- entry
	}
	close(query.response)
	log.Printf("Query: %#v", query)
}
