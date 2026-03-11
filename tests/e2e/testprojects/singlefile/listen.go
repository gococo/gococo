package main

import "net"

func listenOnPort(port string) (net.Listener, error) {
	return net.Listen("tcp", "127.0.0.1:"+port)
}
