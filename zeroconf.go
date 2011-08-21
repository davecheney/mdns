package zeroconf

import (
	"fmt"
	"log"
	"net"

	dns "github.com/miekg/godns"
)

var (
	Registry = newRegistry()
)

func openIPv4Socket() *net.UDPConn {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: 5353,
	})
	if err != nil {
		log.Fatal(err)
	}
	return conn
}

func init() {
	go Registry.mainloop()

	socket := openIPv4Socket()
	if err := socket.JoinGroup(nil, net.IPv4(224, 0, 0, 251)); err != nil {
		log.Fatal(err)
	}

	listener := &listener{
		socket:   socket,
		registry: Registry,
	}
	go listener.mainloop()

	// simple logger
	go func() {
		for op := range Registry.Subscribe() {
			switch op.Op {
			case Add:
				log.Printf("Add: %#v", op.Service)
			case Remove:
				log.Printf("Remove: %#v", op.Service)
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

type Service struct {
	Type      string
	Name      string
	Domain    string
	Interface *net.Interface
	Address   net.Addr
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

type registry struct {
	ops         chan Operation
	subscribe   chan *subscription
	services    []*Service
	subscribers []*subscription
}

// TODO registry is not an exported type, should it be?
func newRegistry() *registry {
	return &registry{
                ops:         make(chan Operation),
		subscribe:	make(chan *subscription),
                services:    nil,
                subscribers: nil,
        }
}

func (r *registry) Add(service *Service) {
	r.ops <- Operation{
		Op:      Add,
		Service: service,
	}
}

func (r *registry) Remove(service *Service) {
	r.ops <- Operation{
		Op:      Remove,
		Service: service,
	}
}

// TODO subscribe should take a *Query
func (r *registry) Subscribe() chan Operation {
	s := &subscription {
		make(chan Operation),
	}
	r.subscribe <- s
	return s.c
}

func (r *registry) mainloop() {
	for {
		select {
		case op := <-r.ops:
			switch op.Op {
			case Add:
				r.addService(op.Service)
			}
			r.notifySubscribers(op)
		case sub := <- r.subscribe:
			r.subscribers = append(r.subscribers, sub)
		}
	}
}

func (r *registry) addService(service *Service) {
	r.services = append(r.services, service)
}

func (r *registry) notifySubscribers(op Operation) {
	for i, _ := range r.subscribers {
		r.subscribers[i].notify(op)
	}
}

type subscription struct {
	c chan Operation
}

func (s *subscription) notify(op Operation) {
	s.c <- op // TODO use non blocking send in case reciver is full
}

type listener struct {
	socket   *net.UDPConn
	registry *registry
}

func (l *listener) mainloop() {
	buf := make([]byte, 1500)
	var msg dns.Msg // TODO should the be reused ?
	for {
		read, err := l.socket.Read(buf)
		if err != nil {
			log.Fatal(err)
		}
		msg.Unpack(buf[:read])
		if msg.Response {
			s := new(Service)
			for _, rr := range msg.Answer {
				log.Printf("%#v", rr)
				switch r := rr.(type) {
				case *dns.RR_SRV:
					
				}
			}
			l.registry.Add(s)
		}
	}
}
