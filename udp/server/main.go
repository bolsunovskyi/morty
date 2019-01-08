package main

import (
	"bytes"
	"flag"
	"io"
	"log"
	"net"
)

var (
	udpPort, tcpPort int
	udpThreads       map[string]chan []byte
	tcpThreads       map[string]chan []byte
)

func init() {
	flag.IntVar(&tcpPort, "tp", 27013, "tcp listen port")
	flag.IntVar(&udpPort, "up", 27013, "udp listen port")
	flag.Parse()

	udpThreads = make(map[string]chan []byte)
	tcpThreads = make(map[string]chan []byte)
}

func tcpListener(conn net.Conn) {
	ip := conn.RemoteAddr().(*net.TCPAddr).IP.String()
	udpThreads[ip] = make(chan []byte, 10)
	tcpThreads[ip] = make(chan []byte, 10)

	go clientListener(ip)

	buffer := make([]byte, 1024)

	for {
		n, err := conn.Read(buffer)
		if err == io.EOF {
			return
		} else if err != nil {
			log.Println(err)
			continue
		}

		if n > 0 {
			tcpThreads[ip] <- buffer[:n]
		}
	}
}

func clientListener(ip string) {
	var (
		udpBuf = new(bytes.Buffer)
	)

	for {
		select {
		case packet := <-udpThreads[ip]:
			udpBuf.Write(packet)
			log.Println("udp", len(packet))
		case <-tcpThreads[ip]:
			udpBuf.Reset()
			log.Println("tcp")
		}
	}
}

func tcpListen() {
	l, err := net.ListenTCP("tcp4", &net.TCPAddr{
		Port: tcpPort,
		IP:   net.IPv4(0, 0, 0, 0),
	})

	if err != nil {
		log.Fatalln(err)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println(err)
		}

		go tcpListener(conn)
	}
}

func udpListen() {
	pc, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: udpPort,
	})
	if err != nil {
		log.Fatalln(err)
	}

	buffer := make([]byte, 1024)

	for {
		n, addr, err := pc.ReadFrom(buffer)
		if err != nil {
			log.Println(err)
		}
		if n > 0 {
			if th, ok := udpThreads[addr.(*net.UDPAddr).IP.String()]; ok {
				th <- buffer[:n]
			}
		}
	}
}

func main() {
	go tcpListen()
	udpListen()
}
