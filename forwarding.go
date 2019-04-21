package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"

	"golang.org/x/net/websocket"
)

func usage() {
	fmt.Printf(`wsforwariding v0.1.0
	wsforwariding -s <listen address> <remote address>
	wsforwariding -c <listen address> <remote address>
`)
}

func main() {
	if len(os.Args) < 4 {
		usage()
		os.Exit(1)
		return
	}
	var err error
	switch os.Args[1] {
	case "-s":
		err = runServer(os.Args[2], os.Args[3])
	case "-c":
		err = runClient(os.Args[2], os.Args[3])
	case "-e":
		err = runEcho(os.Args[2])
	default:
		err = fmt.Errorf("not supported command:%v", os.Args[1])
	}
	fmt.Printf("stopping with error:%v\n", err)
}

func runServer(listen, target string) (err error) {
	fmt.Printf("start server mode by listen:%v,target:%v\n", listen, target)
	targetURL, err := url.Parse(target)
	if err != nil {
		return
	}
	handler := func(ws *websocket.Conn) {
		fmt.Printf("start forwarding to %v from %v\n", target, ws.RemoteAddr())
		conn, err := net.Dial(targetURL.Scheme, targetURL.Host)
		if err != nil {
			fmt.Printf("connect to %v fail with %v\n", targetURL.Host, err)
			ws.Close()
			return
		}
		go func() {
			_, err = io.Copy(conn, ws)
			ws.Close()
		}()
		_, err = io.Copy(ws, conn)
		conn.Close()
	}
	http.Handle("/wsforwarding", websocket.Handler(handler))
	err = http.ListenAndServe(listen, nil)
	return
}

func runClient(listen, target string) (err error) {
	fmt.Printf("start clien mode by listen:%v,target:%v\n", listen, target)
	listenURL, err := url.Parse(listen)
	if err != nil {
		return
	}
	listener, err := net.Listen(listenURL.Scheme, listenURL.Host)
	if err != nil {
		return
	}
	var conn net.Conn
	for {
		conn, err = listener.Accept()
		if err != nil {
			break
		}
		fmt.Printf("accept connection from %v\n", conn.RemoteAddr())
		go func() {
			fmt.Printf("start forwarding to %v from %v\n", target, conn.RemoteAddr())
			var config *websocket.Config
			config, err = websocket.NewConfig(target, "https://wsf.snows.io/")
			if err != nil {
				return
			}
			config.TlsConfig = &tls.Config{InsecureSkipVerify: true}
			remote, err := websocket.DialConfig(config)
			if err != nil {
				fmt.Printf("dail to %v fail with %v\n", target, err)
				conn.Close()
				return
			}
			go procCopy(remote, conn)
			procCopy(conn, remote)
		}()
	}
	return
}

func runEcho(listen string) (err error) {
	listenURL, err := url.Parse(listen)
	if err != nil {
		return
	}
	listener, err := net.Listen(listenURL.Scheme, listenURL.Host)
	if err != nil {
		return
	}
	var conn net.Conn
	for {
		conn, err = listener.Accept()
		if err != nil {
			break
		}
		go procCopy(conn, conn)
	}
	return
}

func procCopy(src io.ReadCloser, dst io.WriteCloser) {
	io.Copy(dst, src)
	src.Close()
	dst.Close()
}
