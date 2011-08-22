package zeroconf

import (
	"fmt"
	"log"
	"net"
)

var (
	ServiceRegistry = &serviceRegistry{
		ops:         make(chan Operation),
		subscribe:   make(chan subscription),
		services:    nil,
		subscribers: nil,
	}
)

func init() {
	go ServiceRegistry.mainloop()

	// simple logger
	go func() {
		for op := range ServiceRegistry.Subscribe() {
			switch op.Op {
			case Add:
				log.Print("Add: ", op.Service)
			case Remove:
				log.Print("Remove: ", op.Service)
			}
		}
	}()
}

func mcastInterfaces() []net.Interface {
	ifaces := make([]net.Interface, 0)
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Fatal(err)
	}
	for _, i := range interfaces {
		if isMulticast(i) {
			fmt.Printf("%#v\n", i)
			ifaces = append(ifaces, i)
		}
	}
	return ifaces
}

func isMulticast(i net.Interface) bool {
	return (i.Flags&net.FlagUp > 0) && (i.Flags&net.FlagMulticast > 0)
}

type op int

const (
	Add    op = 1
	Remove op = 2
)

type Operation struct {
	Op      op
	Service *Service
}

type serviceRegistry struct {
	ops         chan Operation
	subscribe   chan subscription
	services    []*Service
	subscribers []subscription
}

func (r *serviceRegistry) AddService(service *Service) {
	r.ops <- Operation{Add, service}
}

func (r *serviceRegistry) RemoveService(service *Service) {
	r.ops <- Operation{Remove, service}
}

// TODO subscribe should take a *Query
func (r *serviceRegistry) Subscribe() chan Operation {
	s := make(chan Operation)
	r.subscribe <- s
	return s
}

func (r *serviceRegistry) mainloop() {
	for {
		select {
		case op := <-r.ops:
			switch op.Op {
			case Add:
				r.services = append(r.services, op.Service)
			}
			r.notifySubscribers(op)
		case sub := <-r.subscribe:
			r.subscribers = append(r.subscribers, sub)
		}
	}
}

func (r *serviceRegistry) notifySubscribers(op Operation) {
	for i, _ := range r.subscribers {
		//TODO use non blocking send in case reciever is full
		r.subscribers[i] <- op
	}
}

type subscription chan Operation

type Service struct {
	Type       string
	Name       string
	Domain     string
	Host       string
	Port       uint16
	Additional []string
}

func (s *Service) String() string {
	return fmt.Sprintf("%s\t%s.%s\t@%s:%d", s.Name, s.Type, s.Domain, s.Host, s.Port)
}
