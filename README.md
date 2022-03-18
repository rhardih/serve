# serve
Simple http server for localhost development.

It's like `python -m SimpleHTTPServer`, but with support for gzip and HTTP/2.

## Installation

```bash
go get github.com/rhardih/serve
```

## Usage

If $GOPATH/bin is in your $PATH, simply:


```
$ serve -h
NAME:
   serve - deliver content of current directory via http/https

USAGE:
   serve [global options] command [command options] [arguments...]

VERSION:
   1.2.0

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --gzip, -g                enable gzip encoding (default: false)
   --port value, -p value    specify port for listening (default: 8080)
   --logging, -l             enable logging output (default: false)
   --http2, -2               enable http2, this generates a self signed
                             certificate, if one isn't already present; cert.pem,
                             key.pem (default: false)
   --header value, -H value  custom header(s) to add to the response (can be
                             repeated multiple times)
   --help, -h                show help (default: false)
   --version, -v             print the version (default: false)  
```

## Note

Code for generating self-signed certificate for HTTP/2 was taken from example code in `src/crypto/tls/generate_cert.go`, available at [https://golang.org/src/crypto/tls/generate_cert.go](https://golang.org/src/crypto/tls/generate_cert.go).
