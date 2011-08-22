package zeroconf

var (
        Local = NewZone("local.")
)

func init() {
        go Local.mainloop()
	Listen(Local) // start mcast listener
}

