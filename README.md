# serve
Simple http server for localhost development with a flag for enabling gzip

## Installation

```bash
go get github.com/rhardih/serve
```

## Usage

If $GOPATH/bin is in your $PATH, simply:


```bash
$> serve -h

NAME:
   serve - deliver content of current directory via http

USAGE:
   serve [global options] command [command options] [arguments...]
   
VERSION:
   0.0.1
   
COMMANDS:
GLOBAL OPTIONS:
   --gzip, -g		enable gzip encoding
   --port, -p "8080"	specify port for listening
   --logging, -l	enable logging output
   --help, -h		show help
   --version, -v	print the version
   
```
