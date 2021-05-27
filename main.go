package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/gcbb/src/net"
)

func main() {
	selfid, _ := strconv.Atoi(os.Args[1])
	peerid, _ := strconv.Atoi(os.Args[2])
	id := [20]byte{byte(selfid)}
	pid := [20]byte{byte(peerid)}
	fmt.Println(id, pid)
	dht := net.NewKadDHT(pid, 5)
	server := net.NewNaiveP2PServer(id, 3, 5, dht, &net.GobNetEncoder{})
	if id == [20]byte{3} {
		server.Start()
	} else {
		server.Start()
		if selfid < peerid {
			fmt.Println("JOASASJ")
			server.ConnectUDP(pid)
		}
	}

	c := make(chan int, 1)
	<-c
}
