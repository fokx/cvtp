## cvtp
convert proxy

### build
```
go build ./cvtp.go
```

### usage
```
$ ./cvtp  -h
Usage of ./cvtp:
  -bypass
        whether to bypass the socks5 proxy
  -from string
        socks5 addr (default "127.0.0.1:3999")
  -listen string
        listening host (default "127.0.0.1")
  -loglevel string
        log level (debug, info) (default "info")
  -to int
        listening port (default 4099)
```
