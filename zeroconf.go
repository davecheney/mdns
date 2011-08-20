package zeroconf

import (
	"fmt"
	"log"
	"net"
	
	dns "github.com/miekg/godns"
)

var (
	Registry = &registry{
		add:    make(chan *Service),
		remove: make(chan *Service),
		services: make([]*Service,0),
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
	go Registry.mainloop()

	socket := openIPv4Socket()	
	if err := socket.JoinGroup(nil, net.IPv4(224, 0, 0, 251)); err != nil {
		log.Fatal(err)
	}

	listener := &listener {
		socket: socket,
		registry: Registry,
	}
	go listener.mainloop()
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
	return (i.Flags & net.FlagUp > 0) && (i.Flags & net.FlagMulticast > 0)
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

type listener struct {
	socket *net.UDPConn
	registry *registry
}	

func (l *listener) mainloop() {
	buf := make([]byte, 1500)
	for {
		read, err := l.socket.Read(buf)
		if err != nil {
			log.Fatal(err)
		}
		msg := &dns.Msg{}
		msg.Unpack(buf[:read])
		log.Println(msg.String())
	}
}
