Introduction
============

This package allows Go processes to publish multicast DNS style records onto their local network segment. For more information about mDNS, and it's closely related cousin, Zeroconf, please visit http://www.multicastdns.org/.

Acknowledgements
================

Thanks to Brian Ketelsen and Miek Gieben for their feedback and suggestions. This package builds on Miek's fantastic godns library and would not have been possible without it.

Installation
============

This package is goinstall'able.

    goinstall github.com/davecheney/mdns

For development, this package is developed with John Asmuths excellent gb utility.

Usage
=====

Publishing mDNS records is as simple as importing the mdns page

    import (
        "net"	// needed for net.IP		
        "github.com/davecheney/mdns"
    )

Then calling one of the publish functions

    mdns.PublishA("yourhost.local", 3600, net.IP(192,168,1,100))

This places an A record into the internal zone file. Broadcast mDNS queries that match records in the internal zone file are responded to automatically. Other records types are supported, check the godoc for more information.

    godoc githib.com/davecheney/mdns

Tested Platforms
================

This package has been tested on the following platforms

* linux/arm
* linux/386
* darwin/386

gmx Instruments
===============

Counters for zone queries and entries, as well as connecto questions and responses are instrumented via gmx.

	gmxc -p $(pgrep mdns-publisher) mdns | sort
	mdns.connector.questions: 0
	mdns.connector.responses: 0
	mdns.zone.local.entries: 5
	mdns.zone.local.queries: 0

Changelog
=========

07/02/2012

* Updated LICENCE to a proper BSD 2 clause
* Added gmx instrumentation
* Updated to the Go 1 multicast API 

14/10/2011 Initial Release
