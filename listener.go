package zeroconf

import (
	"fmt"
	"log"
	"net"

	dns "github.com/miekg/godns"
)

func Listen(zone *Zone) *listener {
        listener := &listener{
                socket: openIPv4Socket(net.IPv4zero),
                zone:   zone,
        }

        if err := listener.socket.JoinGroup(nil, net.IPv4(224, 0, 0, 251)); err != nil {
                log.Fatal(err)
        }

        go listener.mainloop()
	return listener
}

func openIPv4Socket(ip net.IP) *net.UDPConn {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   ip,
		Port: 5353,
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
	buf := make([]byte, 1500)
	for {
		read, _, err := l.socket.ReadFromUDP(buf)
		if err != nil {
			log.Fatal(err)
		}
		msg := new(dns.Msg)
		msg.Unpack(buf[:read])
		if isQuestion(msg) {
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
					expires: 2 ^ 31,
					publish: false,
					rr:      rr,
				})
			}
		}
	}
}

func (l *listener) SendResponse(answers []dns.RR) {
	response := &dns.Msg {
		MsgHdr: dns.MsgHdr{
			Response: true,
		}, 
		Answer: answers,
	}
  	if buf, ok := response.Pack() ; ok {
       		l.socket.Write(buf) 
        }
}


func isQuestion(msg *dns.Msg) bool {
	return !msg.MsgHdr.Response
}
