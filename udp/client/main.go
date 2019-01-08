package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os/exec"
)

var (
	host             string
	tcpPort, udpPort int
	tcpConn, udpConn net.Conn
	tcpErr           chan error
	stdOutListener   StdOutListener
)

func init() {
	flag.IntVar(&tcpPort, "tp", 27013, "server tcp port")
	flag.IntVar(&udpPort, "up", 27013, "server udp port")
	flag.StringVar(&host, "h", "localhost", "server host")
	flag.Parse()

	tcpErr = make(chan error)

	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

type StdOutListener struct {
	udpConn net.Conn
}

func (s *StdOutListener) Write(p []byte) (n int, err error) {
	_, nErr := s.udpConn.Write(p)
	if nErr != nil {
		tcpErr <- nErr
	}
	return len(p), nil
}

func connect() error {
	var err error
	tcpConn, err = net.Dial("tcp", fmt.Sprintf("%s:%d", host, tcpPort))
	if err != nil {
		return err
	}
	log.Println("tcp connected")

	udpConn, err = net.Dial("udp", fmt.Sprintf("%s:%d", host, udpPort))
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("udp connected")

	stdOutListener.udpConn = udpConn

	return nil
}

func reconnect() {
	for {
		err := <-tcpErr
		if err != nil {
			log.Println(err)
		}

		if err := connect(); err != nil {
			log.Println(err)
		}
	}
}

func main() {
	if err := connect(); err != nil {
		log.Fatalln(err)
	}

	go reconnect()

	for {
		rec := exec.Command("rec", "-c", "1", "-t", "wav", "/dev/stdout", "rate", "16k", "silence", "1", "0.1", "3%", "1", "0.7", "3%")
		rec.Stdout = &stdOutListener
		if err := rec.Run(); err != nil {
			log.Println(err)
			continue
		}

		if tcpConn != nil {
			if _, err := tcpConn.Write([]byte("e")); err != nil {
				log.Println(err)
				tcpErr <- err
				continue
			}
		}
		log.Println("sent")
	}
}
