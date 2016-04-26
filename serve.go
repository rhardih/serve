package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/daaku/go.httpgzip"
	"net/http"
	"os"
)

func main() {
	app := cli.NewApp()

	app.Name = "serve"
	app.Usage = "deliver content of current directory via http"
	app.Version = "0.0.1"

	var gzip bool
	var port int = 8080
	var path string = "./"

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
	}

	app.Action = func(c *cli.Context) {
		if c.NArg() > 0 {
			path = c.Args()[0]
		}

		handler := http.FileServer(http.Dir(path))

		if gzip {
			handler = httpgzip.NewHandler(handler)
		}

		println(fmt.Sprintf("Serving content of %s on localhost:%v ...", path, port))

		http.ListenAndServe(fmt.Sprintf(":%v", port), handler)
	}

	app.Run(os.Args)
}
