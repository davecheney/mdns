package zeroconf

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

	dns "github.com/miekg/godns"
)

const (
	seconds = 1e9
)

var (
	IPv4MCASTADDR = &net.UDPAddr{
		IP:   net.IPv4(224, 0, 0, 251),
		Port: 5353,
	}
)

func Listen(zone *Zone) *listener {
	listener := &listener{
		socket: openIPv4Socket(net.IPv4zero),
		zone:   zone,
	}

	if err := listener.socket.JoinGroup(nil, IPv4MCASTADDR.IP); err != nil {
		log.Fatal(err)
	}

	go listener.mainloop()
	return listener
}

func openIPv4Socket(ip net.IP) *net.UDPConn {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   ip,
		Port: IPv4MCASTADDR.Port,
	})
	if err != nil {
		log.Fatal(err)
	}
	return conn
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

type listener struct {
	socket *net.UDPConn
	zone   *Zone
}

func (l *listener) mainloop() {
	for {
		msg, err := l.readMessage()
		if err != nil {
			log.Fatal(err)
		}
		if msg.IsQuestion() {
			var answers []dns.RR
			for _, question := range msg.Question {
				for result := range l.zone.Query(question) {
					if result.publish {
						answers = append(answers, result.rr)
					}
				}
			}
			l.SendResponse(answers)
		} else {
			for _, rr := range msg.Answer {
				l.zone.Add(&Entry{
					expires: time.Nanoseconds() + int64(rr.Header().Ttl*seconds),
					publish: false,
					rr:      rr,
				})
			}
		}
	}
}

func (l *listener) SendResponse(answers []dns.RR) {
	response := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Response: true,
		},
		Answer: answers,
	}
	if err := l.writeMessage(response); err != nil {
		log.Fatal(err)
	}
}

func (l *listener) writeMessage(msg *dns.Msg) (err os.Error) {
	if buf, ok := msg.Pack(); ok {
		_, err = l.socket.WriteToUDP(buf, IPv4MCASTADDR)

	}
	return
}

func (l *listener) readMessage() (*dns.Msg, os.Error) {
	buf := make([]byte, 1500)
	read, err := l.socket.Read(buf)
	if err != nil {
		return nil, err
	}
	msg := new(dns.Msg)
	if msg.Unpack(buf[:read]) {
		return msg, nil
	}
	return nil, os.NewError("Unable to unpack buffer")

}
