Introduction
============

This package allows Go processes to publish multicast DNS style records onto their local network segment. For more information about mDNS, and it's closely related cousin, Zeroconf, please visit http://www.multicastdns.org/.

Acknowledgements
================

Thanks to Brian Ketelsen and Miek Gieben for their feedback and suggestions. This package builds on Miek's fantastic godns library and would not have been possible without it.

Installation
============

This package can be installed using:

    go get github.com/davecheney/mdns

For development, this package is developed with John Asmuths excellent gb utility.

Usage
=====

Publishing mDNS records is simple

    import "github.com/davecheney/mdns"

    mdns.Publish("yourhost.local 60 IN A 192.168.1.100")

This places an A record into the internal zone file. Broadcast mDNS queries that match records in the internal zone file are responded to automatically. Other records types are supported, check the godoc for more information.

    go doc github.com/davecheney/mdns

Tested Platforms
================

This package has been tested on the following platforms

* linux/arm
* linux/386
* linux/amd64
* darwin/386
* darwin/amd64

gmx Instruments
===============

Counters for zone queries and entries, as well as connector questions and responses are instrumented via gmx.

	gmxc -p $(pgrep mdns-publisher) mdns | sort
	mdns.connector.questions: 3
	mdns.connector.responses: 3
	mdns.zone.local.entries: 5
	mdns.zone.local.queries: 12

Changelog
=========

12/02/2012

* Simplified mdns.Publish method, thanks to Miek Gieben for his quick work adding parsing support for SRV and PTR records.

07/02/2012

* Updated LICENCE to a proper BSD 2 clause
* Added gmx instrumentation
* Updated to the Go 1 multicast API 

14/10/2011 Initial Release
