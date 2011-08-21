package zeroconf

import (
	"fmt"
	"log"
	"net"
	"strings"

	dns "github.com/miekg/godns"
)

var (
	ServiceRegistry = &serviceRegistry{
		ops:         make(chan Operation),
		subscribe:   make(chan subscription),
		services:    nil,
		subscribers: nil,
	}
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
	go ServiceRegistry.mainloop()

	socket := openIPv4Socket()
	if err := socket.JoinGroup(nil, net.IPv4(224, 0, 0, 251)); err != nil {
		log.Fatal(err)
	}

	listener := &listener{
		socket:          socket,
		serviceRegistry: ServiceRegistry,
	}
	go listener.mainloop()

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

type listener struct {
	socket          *net.UDPConn
	serviceRegistry *serviceRegistry
}

func (l *listener) mainloop() {
	buf := make([]byte, 1500)
	for {
		read, _, err := l.socket.ReadFromUDP(buf)
		if err != nil {
			log.Fatal(err)
		}
		msg := new(dns.Msg)
		msg.Unpack(buf[:read])
		switch msg.MsgHdr.Response {
		case true:
			s := new(Service)
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
			if len(s.Name) > 0 {
				l.serviceRegistry.AddService(s)
			}
		case false:
			for _, rr := range msg.Question {
                                        log.Printf("%#v", rr)
			}
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

func (s *Service) String() string {
	return fmt.Sprintf("%s\t%s.%s\t@%s:%d", s.Name, s.Type, s.Domain, s.Host, s.Port)
}

// Publish a service by inserting the record into the registry
// then sending a query for the record
func Publish(s *Service) {

}
