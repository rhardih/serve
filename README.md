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
$> serve -h

NAME:
   serve - deliver content of current directory via http

USAGE:
   serve [global options] command [command options] [arguments...]
   
VERSION:
   1.0.0
   
COMMANDS:
GLOBAL OPTIONS:
   --gzip, -g		enable gzip encoding
   --port, -p "8080"	specify port for listening
   --logging, -l	enable logging output
   --http2, -2		enable http2, this generates a self signed certificate, if one isn't already present; cert.pem, key.pem
   --help, -h		show help
   --version, -v	print the version
   
```

## Note

Code for generating self-signed certificate for HTTP/2 was taken from example code in `src/crypto/tls/generate_cert.go`, available at [https://golang.org/src/crypto/tls/generate_cert.go](https://golang.org/src/crypto/tls/generate_cert.go).
