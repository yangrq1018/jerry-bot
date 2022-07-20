package emby

import (
	"fmt"
	"log"
	"net"
)

func DiscoverEmbyServer() (string, error) {
	// 这里设置接收者的IP地址为广播地址
	srcAddr := &net.UDPAddr{
		Port: 50897,
	}
	dstAddr := &net.UDPAddr{
		IP:   net.IPv4(10, 168, 1, 255),
		Port: 7359,
	}
	conn, err := net.DialUDP("udp", srcAddr, dstAddr)
	if err != nil {
		return "", err
	}
	_, err = fmt.Fprint(conn, "who is EmbyServer?")
	if err != nil {
		return "", err
	}
	log.Println("message send")
	buffer := make([]byte, 2028)
	for {
		i, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Println(err)
			break
		}
		log.Printf("receive from %v, content: %s\n", addr, string(buffer[:i]))
	}
	return "", nil
}
