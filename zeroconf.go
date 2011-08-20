package zeroconf

import (
	"net"
)

var (
	Registry = &registry{
		add:    make(chan *Service),
		remove: make(chan *Service),
		services: make([]*Service,0),
	}
)

func init() {
	go Registry.mainloop()
}

type Service struct {
	Type      string
	Name      string
	Domain    string
	Interface *net.Interface
	Address   net.Addr
}

type registry struct {
	add, remove chan *Service
	services []*Service
}

func (r *registry) Add(service *Service) {
	r.add <- service
}

func (r *registry) mainloop() {
	for {
		select {
		case s := <- r.add:
			r.services = append(r.services, s)
		}
	}
}
