package zeroconf

import (
	"fmt"
	"log"
	"net"
	"strings"

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
	subscribe   chan subscription
	services    []*Service
	hosts       []*Host
	subscribers []subscription
}

// TODO registry is not an exported type, should it be?
func newRegistry() *registry {
	return &registry{
		ops:         make(chan Operation),
		subscribe:   make(chan subscription),
		services:    nil,
		hosts:       nil,
		subscribers: nil,
	}
}

func (r *registry) AddService(service *Service) {
	r.ops <- Operation{
		Op:      Add,
		Service: service,
	}
}

func (r *registry) RemoveService(service *Service) {
	r.ops <- Operation{
		Op:      Remove,
		Service: service,
	}
}

// TODO subscribe should take a *Query
func (r *registry) Subscribe() chan Operation {
	s := make(chan Operation)
	r.subscribe <- s
	return s
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
		case sub := <-r.subscribe:
			r.subscribers = append(r.subscribers, sub)
		}
	}
}

func (r *registry) addService(service *Service) {
	r.services = append(r.services, service)
}

func (r *registry) notifySubscribers(op Operation) {
	for i, _ := range r.subscribers {
		//TODO use non blocking send in case reciever is full
		r.subscribers[i] <- op
	}
}

type subscription chan Operation

type listener struct {
	socket   *net.UDPConn
	registry *registry
}

func (l *listener) mainloop() {
	buf := make([]byte, 1500)
	for {
		read, err := l.socket.Read(buf)
		if err != nil {
			log.Fatal(err)
		}
		msg := new(dns.Msg)
		msg.Unpack(buf[:read])
		s := new(Service)
		s.unmarshal(msg)
		if s.valid() {
			l.registry.AddService(s)
		}
	}
}

type Service struct {
	Type       string
	Name       string
	Domain     string
	Host       string
	Port       uint16
	Additional []string
}

// s.unmarshal may not be complete, return false if so
func (s *Service) valid() bool {
	return len(s.Name) > 0
}

func (s *Service) unmarshal(msg *dns.Msg) {
	for i := range msg.Answer {
		switch rr := msg.Answer[i].(type) {
		case *dns.RR_SRV:
			x := strings.Split(rr.Hdr.Name, ".")
			s.Name = x[0]
			s.Type = strings.Join(x[1:3], ".")
			s.Domain = strings.Join(x[3:], ".")
			s.Host = rr.Target
			s.Port = rr.Port

		case *dns.RR_TXT:
			s.Additional = append(s.Additional, strings.Split(rr.Txt, ",")...)
		default:
			log.Printf("%#v", rr)
		}
	}
}

type Host struct {
	Addrs []*net.Addr
	Name  string
}
