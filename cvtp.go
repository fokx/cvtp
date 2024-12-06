package main

import (
	"flag"
	"golang.org/x/net/proxy"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var logLevel string

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

func serveHTTP(w http.ResponseWriter, req *http.Request, socks5Addr string, bypassPtr bool, proxyList []string) {
	d := &net.Dialer{
		Timeout: 10 * time.Second,
	}
	var dialer proxy.Dialer = proxy.Direct
	if !bypassPtr {
		real_socks5Addr := socks5Addr
		if len(proxyList) != 0 {
			n := rand.Int() % len(proxyList)
			real_socks5Addr := proxyList[n]
			if logLevel == "debug" {
				log.Printf("Using upstream %s", real_socks5Addr)
			}
		}
		dialer, _ = proxy.SOCKS5("tcp", real_socks5Addr, nil, d)
	}

	if req.Method == "CONNECT" {
		handleTunnel(w, req, dialer)
	} else {
		handleHTTP(w, req, dialer)
	}
}

type proxyStringList []string

func (s *proxyStringList) String() string {
	return strings.Join(*s, ",")
}

func (s *proxyStringList) Set(value string) error {
	if value == "" {
		*s = []string{}
	} else {
		tmp := strings.Split(value, ",")
		res := make([]string, 0, len(tmp))
		for i := range tmp {
			item := strings.TrimSpace(tmp[i])
			if item == "" {
				continue
			}
			if _, err := strconv.Atoi(item); err == nil {
				// if it's a number, assume it's a port number on localhost
				item = "127.0.0.1:" + item
			}
			res = append(res, item)
		}
		*s = res
	}
	return nil
}

func main() {
	var proxylist proxyStringList

	logLevelPtr := flag.String("loglevel", "info", "log level (debug, info)")
	listenHostPtr := flag.String("host", "127.0.0.1", "listen on host (ip / domain). if set to 'all', will listen on all interfaces (0.0.0.0)")
	listenPortPtr := flag.Int("port", 4099, "listen on port")
	socks5ProxyAddrPtr := flag.String("from", "127.0.0.1:3999", "upstream socks5 addr")
	bypassPtr := flag.Bool("bypass", false, "whether to bypass the socks5 proxy. will ignore '-from' if set")
	flag.Var(&proxylist, "fromlist", "comma-separated list of proxy upstreams, each will be used randomly for each new connection. \nwill ignore '-from' if set. (default to not set)\nfor each item, will assume it's a port number on localhost if it's a number. \ne.g. 1080,1081,1082 (all localhost ports) or 1080,192.168.1.1:1081,127.0.0.1:5606")
	flag.Parse()

	if *listenHostPtr == "all" {
		*listenHostPtr = "0.0.0.0"
	}
	logLevel = *logLevelPtr
	log.Printf("Listening on %s:%d", *listenHostPtr, *listenPortPtr)
	log.Printf("Log level: %s", logLevel)
	if *bypassPtr {
		log.Printf("Bypassing the socks5 proxy")
	} else {
		log.Printf("Using the socks5 proxy at %s", *socks5ProxyAddrPtr)
	}
	if len(proxylist) != 0 {
		log.Printf("Upstream proxy list: %v", proxylist)
	}
	err := http.ListenAndServe(*listenHostPtr+":"+strconv.Itoa(*listenPortPtr), http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if logLevel == "debug" {
			log.Printf("Request target address: %s", req.Host)
		}
		serveHTTP(w, req, *socks5ProxyAddrPtr, *bypassPtr, proxylist)
	}))
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
