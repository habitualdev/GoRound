package main

import (
	"fmt"
	"github.com/1lann/udp-forward"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	meltCheck = true
	localAddr = net.IP{0, 0, 0, 0}
	loopCheck = true
	n         = 1
)

var helpMessage = `Usage:
-u localPort:remoteAddress:remotePort				UDP proxy
-t localPort:remoteAddress:remotePort				TCP proxy
-l 0.0.0.0											Change local address binding (default: 0.0.0.0)
-m													Melts the binary
-h/--help											Print this message

Multiple options can be chained as desired.`

func joinAddr(addr net.IP, port int) string {
	return addr.String() + ":" + strconv.Itoa(port)
}

func copyIO(src, dest net.Conn) {
	defer src.Close()
	defer dest.Close()
	io.Copy(src, dest)
}

func tcpRedirect(localAddr net.IP, localPort int, remoteAddr net.IP, remotePort int) {
	listener, err := net.Listen("tcp", joinAddr(localAddr, localPort))
	if err != nil {
		panic(err)
	}
	for {
		conn, _ := listener.Accept()

		go func() {
			conn2, _ := net.Dial("tcp", joinAddr(remoteAddr, remotePort))
			go copyIO(conn2, conn)
			go copyIO(conn, conn2)
		}()
	}
}

func udpRedirect(localAddr net.IP, localPort int, remoteAddr net.IP, remotePort int) {
	forward.Forward(joinAddr(localAddr, localPort), joinAddr(remoteAddr, remotePort), forward.DefaultTimeout)
	for loopCheck {
		time.Sleep(1 * time.Millisecond)
	}
}

func cleanup() {
	loopCheck = false
	time.Sleep(3 * time.Second)
	os.Exit(1)
}

func paramRecover(args []string) {
	if err := recover(); err != nil {
		fmt.Println("Expected parameter not found for " + args[n-1])
		fmt.Println("Exiting...")
		os.Exit(1)
		return
	}
}

func parseRecover(raw string) {
	if err := recover(); err != nil {
		fmt.Println("Unable to parse " + raw)
		fmt.Println("Exiting...")
		os.Exit(1)
		return
	}
}

func unmarshalInput(raw string) (net.IP, int, net.IP, int) {
	defer parseRecover(raw)
	sliced := strings.Split(raw, ":")
	if len(sliced) != 3 {
		fmt.Println("Unable to parse \"" + raw + "\"")
		os.Exit(1)
	}

	localPort, _ := strconv.Atoi(sliced[0])
	remoteIP := net.ParseIP(sliced[1])
	remotePort, _ := strconv.Atoi(sliced[2])

	return localAddr, localPort, remoteIP, remotePort
}

func main() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cleanup()
	}()

	args := os.Args

	if args[1] == "-h" || args[1] == "--help" {
		fmt.Println(helpMessage)
		os.Exit(0)
	}

	for len(args) > n {
		defer paramRecover(args)
		if args[n] == "-t" {
			n += 1
			listenAddr, localPort, remoteIP, remotePort := unmarshalInput(args[n])
			n += 1
			go tcpRedirect(listenAddr, localPort, remoteIP, remotePort)

		} else if args[n] == "-u" {
			n += 1
			listenAddr, localPort, remoteIP, remotePort := unmarshalInput(args[n])
			n += 1
			go udpRedirect(listenAddr, localPort, remoteIP, remotePort)
		} else if args[n] == "-l" {
			n += 1
			localAddr = net.ParseIP(args[n])
			n += 1
		} else if args[n] == "-m" {
			n += 1
			if meltCheck {
				fileName, _ := os.Executable()
				os.Remove(fileName)
				meltCheck = false
			}
		} else {
			fmt.Println("Unrecognized parameter " + args[n] + ", exitting")
			os.Exit(1)
		}
	}
	for loopCheck {
		time.Sleep(1 * time.Millisecond)
	}

}
