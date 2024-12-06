## cvtp
convert socks5 proxy to http(s) proxy

### build
```
go build ./cvtp.go
# strip symbols and debug info
go build -ldflags "-s -w" ./cvtp.go 
```

### usage
```
Usage of ./cvtp:
  -from string
        upstream socks5 addr (default "127.0.0.1:3999")
  -fromlist value
        comma-separated list of proxy upstreams, each will be used randomly for each new connection. 
        will ignore '-from' if set. (default to not set)
        for each item, will assume it's a port number on localhost if it's a number. 
        e.g. 1080,1081,1082 (all localhost ports) or 1080,192.168.1.1:1081,127.0.0.1:5606
  -bypass
        whether to bypass the socks5 proxy. will ignore '-from' if set (default to not set)
  -host string
        listen on host (ip / domain). if set to 'all', will listen on all interfaces (0.0.0.0) (default "127.0.0.1")
  -port int
        listen on port (default 4099)
 -loglevel string
    log level (debug, info) (default "info")
```

### Examples 
```
# convert socks5 proxy on 127.0.0.1:1080 to http proxy on 127.0.0.1:1081
./cvtp -from 1080 -host 1081
# convert socks5 proxy on 192.168.1.1:1080 to http proxy on 127.0.0.2:1081
./cvtp -from 192.168.1.1:1080 -host 127.0.0.2 -port 1081
# convert socks5 proxy on 192.168.1.1:1080 to http proxy on 0.0.0.0:1081
./cvtp -from 192.168.1.1:1080 -host all
# convert socks5 proxy on 1080,1081,1082 to http proxy on 127.0.0.1:8888
./cvtp -from 1080,1081,1082 -port 8888
# convert socks5 proxy on 192.168.1.1:1080,127.0.0.1:1081,127.0.0.1:1082 to http proxy on 127.0.0.1:8888
./cvtp -from 192.168.1.1:1080,1081,1082 -port 8888
```