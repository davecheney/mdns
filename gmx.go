package main

import gmx "github.com/davecheney/gmx"

func init() {
	gmx.Publish("mdns.local.queries", func() interface{} {
		return local.queryCount
	})
	gmx.Publish("mdns.local.entries", func() interface{} {
		return local.entryCount
	})
}
