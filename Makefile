include $(GOROOT)/src/Make.inc

TARG=github.com/davecheney/zeroconf
GOFILES=\
	listener.go\
	zone.go\

include $(GOROOT)/src/Make.pkg

