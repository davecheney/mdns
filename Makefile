include $(GOROOT)/src/Make.inc

TARG=github.com/davecheney/zeroconf
GOFILES=\
	connector.go\
	zeroconf.go\
	zone.go\

include $(GOROOT)/src/Make.pkg

