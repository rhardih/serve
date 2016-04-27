package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/daaku/go.httpgzip"
	"log"
	"net/http"
	"os"
)

func logHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.Method, r.RequestURI)
		h.ServeHTTP(w, r)
	})
}

func main() {
	app := cli.NewApp()

	app.Name = "serve"
	app.Usage = "deliver content of current directory via http"
	app.Version = "0.0.1"

	var gzip bool
	var port int
	var path string = "./"
	var logging bool

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:        "gzip, g",
			Usage:       "enable gzip encoding",
			Destination: &gzip,
		},
		cli.IntFlag{
			Name:        "port, p",
			Usage:       "specify port for listening",
			Value:       8080,
			Destination: &port,
		},
		cli.BoolFlag{
			Name:        "logging, l",
			Usage:       "enable logging output",
			Destination: &logging,
		},
	}

	app.Action = func(c *cli.Context) {
		if c.NArg() > 0 {
			path = c.Args()[0]
		}

		handler := http.FileServer(http.Dir(path))

		if gzip {
			handler = httpgzip.NewHandler(handler)
		}

		if logging {
			handler = logHandler(handler)
		}

		log.Println(fmt.Sprintf("Serving content of %s on localhost:%v ...", path, port))

		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", port), handler))
	}

	app.Run(os.Args)
}
