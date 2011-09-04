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
		IP:   net.ParseIP("224.0.0.251"),
		Port: 5353,
	}

	IPv6MCASTADDR = &net.UDPAddr{
		IP:   net.ParseIP("ff02::fb"),
		Port: 5353,
	}
)

type listener struct {
	addr    *net.UDPAddr
	conn    *net.UDPConn
	add     chan *Entry // send entries to zone
	query   chan *Query // send questions to zone
	publish chan *dns.Msg
}

func listen(addr *net.UDPAddr, add chan *Entry, query chan *Query, publish chan *dns.Msg) os.Error {
	conn, err := openSocket(addr)
	if err != nil {
		return err
	}
	if err := conn.JoinGroup(nil, addr.IP); err != nil {
		return err
	}
	l := &listener{
		addr:    addr,
		conn:    conn,
		add:     add,
		query:   query,
		publish: publish,
	}
	go l.mainloop()
	go l.publisher()
	return nil
}

func openSocket(addr *net.UDPAddr) (*net.UDPConn, os.Error) {
	switch addr.IP.To4() {
	case nil:
		return net.ListenUDP("udp6", &net.UDPAddr{
			IP:   net.IPv6zero,
			Port: addr.Port,
		})
	default:
		return net.ListenUDP("udp4", &net.UDPAddr{
			IP:   net.IPv4zero,
			Port: addr.Port,
		})
	}
	panic("unreachable")
}

func (l *listener) mainloop() {
	for {
		msg, addr, err := l.readMessage()
		if err != nil {
			log.Fatalf("Cound not read from %s: %s", l.conn, err)
		}
		if msg.IsQuestion() {
			var answers []dns.RR
			fmt.Println(msg)
			for _, question := range msg.Question {
				results := make(chan *Entry, 16)
				l.query <- &Query{question, results}
				for result := range results {
					if result.Publish {
						answers = append(answers, result.RR)
					}
				}
			}
			l.SendResponse(msg.Question, answers)
		} else {
			for _, rr := range msg.Answer {
				l.add <- &Entry{
					Expires: time.Nanoseconds() + int64(rr.Header().Ttl*seconds),
					Publish: false,
					RR:      rr,
					Source:  addr,
				}
			}
		}
	}
}

func (l *listener) SendResponse(q []dns.Question, answers []dns.RR) {
	if len(answers) > 0 {
		msg := new(dns.Msg)
		msg.MsgHdr.Response = true
		msg.Question = q
		msg.Answer = answers
		l.publish <- msg
	}
}

func (l *listener) writeMessage(msg *dns.Msg) (err os.Error) {
	if buf, ok := msg.Pack(); ok {
		_, err = l.conn.WriteToUDP(buf, l.addr)
	}
	return
}

func (l *listener) publisher() {
	for msg := range l.publish {
		if err := l.writeMessage(msg); err != nil {
			log.Fatalf("Cannot send: %s", err)
		}
	}
	panic("publisher exited")
}

func (l *listener) readMessage() (*dns.Msg, *net.UDPAddr, os.Error) {
	buf := make([]byte, 1500)
	read, addr, err := l.conn.ReadFromUDP(buf)
	if err != nil {
		return nil, nil, err
	}
	if msg := new(dns.Msg); msg.Unpack(buf[:read]) {
		return msg, addr, nil
	}
	return nil, addr, os.NewError("Unable to unpack buffer")
}
