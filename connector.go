package zeroconf

import (
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
		IP:   net.ParseIP("224.0.0.251"),
		Port: 5353,
	}

	IPv6MCASTADDR = &net.UDPAddr{
		IP: net.ParseIP("ff02::fb"),
		Port: 5353,
	}
)

type listener struct {
        conn *net.UDPConn
        add   chan *Entry
	query	chan *Query
}

func listen(conn *net.UDPConn, add chan *Entry, query chan *Query) os.Error {
	if err := conn.JoinGroup(nil, IPv4MCASTADDR.IP); err != nil {
		return err
	}
	l := &listener{
                conn: conn,
                add:   add,
		query: query,
        }
	go l.mainloop()
	return nil
}

func openSocket(addr *net.UDPAddr) *net.UDPConn {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr {
		IP: net.IPv4zero,
		Port: addr.Port,
	})
	if err != nil {
		log.Fatalf("Could not join %s: %s", addr, err)
	}
	return conn
}

func (l *listener) mainloop() {
	for {
		msg, err := l.readMessage()
		if err != nil {
			log.Fatalf("Cound not read from %s: %s", l.conn, err)
		}
		if msg.IsQuestion() {
			var answers []dns.RR
			for _, question := range msg.Question {
				results := make(chan *Entry, 16)
				l.query <- &Query{ question, results }
				for result := range results{
					if result.Publish {
						answers = append(answers, result.RR)
					}
				}
			}
			l.SendResponse(answers)
		} else {
			for _, rr := range msg.Answer {
				l.add <- &Entry{
					Expires: time.Nanoseconds() + int64(rr.Header().Ttl*seconds),
					Publish: false,
					RR:      rr,
				}
			}
		}
	}
}

func (l *listener) SendResponse(answers []dns.RR) {
	if len(answers) > 0 {
		msg := new(dns.Msg)
		msg.MsgHdr.Response = true
		msg.Answer = answers
		if err := l.writeMessage(msg); err != nil {
			log.Fatalf("Cannot send: %s", err)
		}
	}
}

func (l *listener) writeMessage(msg *dns.Msg) (err os.Error) {
	if buf, ok := msg.Pack(); ok {
		_, err = l.conn.WriteToUDP(buf, IPv4MCASTADDR)
	}
	return
}

func (l *listener) readMessage() (*dns.Msg, os.Error) {
	buf := make([]byte, 1500)
	read, _, err := l.conn.ReadFromUDP(buf)
	if err != nil {
		return nil, err
	}
	msg := new(dns.Msg)
	if msg.Unpack(buf[:read]) {
		return msg, nil
	}
	return nil, os.NewError("Unable to unpack buffer")
}
