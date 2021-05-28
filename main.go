package main

import (
	"fmt"
	"net"
	"os"
	"strconv"

	gnet "github.com/gcbb/src/net"
)

func main() {
	selfid, _ := strconv.Atoi(os.Args[1])
	peerid, _ := strconv.Atoi(os.Args[2])
	id := [20]byte{byte(selfid)}
	pid := [20]byte{byte(peerid)}
	fmt.Println(id, pid)
	dht := gnet.NewKadDHT(pid, 5)
	server := gnet.NewNaiveP2PServer(id, 3, 5, dht, &gnet.GobNetEncoder{}, 8090)
	if id == [20]byte{3} {
		server.Start()
	} else {
		pInfo := &gnet.PeerInfo{
			Id: [20]byte{3},
			Ip: &net.UDPAddr{IP: net.IPv4(123, 60, 211, 219), Port: 8090},
		}
		server.Init([]*gnet.PeerInfo{pInfo})
		server.Start()
		// if selfid < peerid {
		// 	fmt.Println("JOASASJ")
		// 	server.ConnectUDP(pid)
		// }
	}

	c := make(chan int, 1)
	<-c
}
