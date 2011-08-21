package zeroconf

import (
	"net"

	dns "github.com/miekg/godns"
)

type Service struct {
        Type      string
        Name      string
        Domain    string
        Interface *net.Interface
        Address   net.Addr
}

// s.unmarshal may not be complete, return false if so
func (s *Service) valid() bool {
	return true
}

func (s *Service) unmarshal(msg *dns.Msg) {

}
