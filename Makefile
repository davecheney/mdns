include $(GOROOT)/src/Make.inc

TARG=github.com/davecheney/zeroconf
GOFILES=\
	service.go\
	zeroconf.go\

include $(GOROOT)/src/Make.pkg

