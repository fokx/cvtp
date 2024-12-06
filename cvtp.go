package main

import (
	"flag"
	"golang.org/x/net/proxy"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
)

func handleHTTP(w http.ResponseWriter, req *http.Request, dialer proxy.Dialer) {
	tp := http.Transport{
		Dial: dialer.Dial,
	}
	resp, err := tp.RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func handleTunnel(w http.ResponseWriter, req *http.Request, dialer proxy.Dialer) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	srcConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	dstConn, err := dialer.Dial("tcp", req.Host)
	if err != nil {
		srcConn.Close()
		return
	}

	srcConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	go transfer(dstConn, srcConn)
	go transfer(srcConn, dstConn)
}

func transfer(dst io.WriteCloser, src io.ReadCloser) {
	defer dst.Close()
	defer src.Close()

	io.Copy(dst, src)
}

func serveHTTP(w http.ResponseWriter, req *http.Request, socks5Addr string, bypassPtr bool) {
	d := &net.Dialer{
		Timeout: 10 * time.Second,
	}
	dialer, _ := proxy.SOCKS5("tcp", socks5Addr, nil, d)
	if bypassPtr {
		dialer = proxy.Direct
	}
	if req.Method == "CONNECT" {
		handleTunnel(w, req, dialer)
	} else {
		handleHTTP(w, req, dialer)
	}
}

func main() {
	listenHostPtr := flag.String("listen", "127.0.0.1", "listening host")
	listeningPortPtr := flag.Int("to", 4099, "listening port")
	socks5AddrPtr := flag.String("from", "127.0.0.1:3999", "socks5 addr")
	bypassPtr := flag.Bool("bypass", false, "whether to bypass the socks5 proxy")
	logLevelPtr := flag.String("loglevel", "info", "log level (debug, info)")
	flag.Parse()

	if *listenHostPtr == "all" {
		*listenHostPtr = "0.0.0.0"
	}
	log.Printf("Listening on %s:%d", *listenHostPtr, *listeningPortPtr)
	log.Printf("Log level: %s", *logLevelPtr)
	if *bypassPtr {
		log.Printf("Bypassing the socks5 proxy")
	} else {
		log.Printf("Using the socks5 proxy at %s", *socks5AddrPtr)
	}
	err := http.ListenAndServe(*listenHostPtr+":"+strconv.Itoa(*listeningPortPtr), http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if *logLevelPtr == "debug" {
			log.Printf("Request target address: %s", req.Host)
		}
		serveHTTP(w, req, *socks5AddrPtr, *bypassPtr)
	}))
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
