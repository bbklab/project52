package main

import (
	"fmt"
	"net"
)

func main() {
	addrs := [][2]string{
		[2]string{"tcp", ":80"},
		[2]string{"tcp", ":22"},
		[2]string{"tcp", ":81"},
		[2]string{"udp", "0.0.0.0:53"},
		[2]string{"udp", "127.0.0.1:1299"},
	}

	for _, addr := range addrs {
		ok := IsAddressListenable(addr[0], addr[1])
		fmt.Println(addr, " --> ", ok)
	}
}

// IsAddressListenable detect if given network:address has been occupied for listening
func IsAddressListenable(network, address string) bool {
	switch network {

	case "tcp", "tcp4", "tcp6": // tcp networks
		addr, err := net.ResolveTCPAddr(network, address)
		if err != nil {
			return false
		}
		l, err := net.ListenTCP(network, addr)
		if err != nil {
			return false
		}
		l.Close()
		return true

	case "udp", "udp4", "udp6": // udp networks
		addr, err := net.ResolveUDPAddr(network, address)
		if err != nil {
			return false
		}
		l, err := net.ListenUDP(network, addr)
		if err != nil {
			return false
		}
		l.Close()
		return true
	}

	return false
}
