package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/gcbb/src/common"
	gnet "github.com/gcbb/src/net"
)

func main() {
	selfid, _ := strconv.Atoi(os.Args[2])
	peerid, _ := strconv.Atoi(os.Args[3])
	staticPort, _ := strconv.Atoi(os.Args[1])
	id := [20]byte{byte(selfid)}
	pid := [20]byte{byte(peerid)}
	fmt.Println(id, pid)
	dht := gnet.NewKadDHT(id, 5)
	server := gnet.NewNaiveP2PServer(id, 3, 5, dht, &common.NaiveNetEncoder{}, staticPort)
	if id == [20]byte{3} {
		server.Start()
	} else {
		pInfo := &gnet.PeerInfo{
			PeerId: [20]byte{3},
			PeerIp: &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8090},
			// PeerIp: &net.UDPAddr{IP: net.IPv4(123, 60, 211, 219), Port: 8090},
		}
		server.Init([]*gnet.PeerInfo{pInfo})
		server.Start()

		if selfid < peerid {
			fmt.Println("JOASASJ")
			tmer := time.NewTimer(20 * time.Second)
			<-tmer.C
			server.ConnectUDP(pid)
		}
	}

	c := make(chan int, 1)
	<-c
}
