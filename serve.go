package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	httpgzip "github.com/daaku/go.httpgzip"
	"github.com/urfave/cli/v2"

	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"strings"
	"time"
)

func logHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.Method, r.RequestURI)
		h.ServeHTTP(w, r)
	})
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to marshal ECDSA private key: %v", err)
			os.Exit(2)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	default:
		return nil
	}
}

// This is extracted from src/crypto/tls/generate_cert.go, keeping only the
// bare minimum needed to create a usable cert for localhost development.
func generateSelfSignedCert() {
	host := "localhost"
	validFor := 365 * 24 * time.Hour
	rsaBits := 2048

	var priv interface{}
	var err error
	priv, err = rsa.GenerateKey(rand.Reader, rsaBits)

	if err != nil {
		log.Fatalf("failed to generate private key: %s", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(validFor)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("failed to generate serial number: %s", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	hosts := strings.Split(host, ",")
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}

	certOut, err := os.Create("cert.pem")
	if err != nil {
		log.Fatalf("failed to open cert.pem for writing: %s", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()
	log.Print("written cert.pem\n")

	keyOut, err := os.OpenFile("key.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Print("failed to open key.pem for writing:", err)
		return
	}
	pem.Encode(keyOut, pemBlockForKey(priv))
	keyOut.Close()
	log.Print("written key.pem\n")
}

func main() {
	var gzip bool
	var port int
	var path string = "./"
	var logging bool
	var http2 bool

	app := &cli.App{
		Name:    "serve",
		Usage:   "deliver content of current directory via http/https",
		Version: "1.1.0",

		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "gzip, g",
				Usage:       "enable gzip encoding",
				Destination: &gzip,
			},
			&cli.IntFlag{
				Name:        "port, p",
				Usage:       "specify port for listening",
				Value:       8080,
				Destination: &port,
			},
			&cli.BoolFlag{
				Name:        "logging, l",
				Usage:       "enable logging output",
				Destination: &logging,
			},
			&cli.BoolFlag{
				Name:        "http2, 2",
				Usage:       "enable http2, this generates a self signed certificate, if one isn't already present; cert.pem, key.pem",
				Destination: &http2,
			},
		},

		Action: func(c *cli.Context) error {
			if c.NArg() > 0 {
				path = c.Args().Get(0)
			}

			handler := http.FileServer(http.Dir(path))

			if gzip {
				handler = httpgzip.NewHandler(handler)
			}

			if logging {
				handler = logHandler(handler)
			}

			log.Println(fmt.Sprintf("Serving content of %s on localhost:%v ...", path, port))

			var err error

			if http2 {
				if _, err := os.Stat("cert.pem"); os.IsNotExist(err) {
					generateSelfSignedCert()
				}

				err = http.ListenAndServeTLS(fmt.Sprintf(":%v", port), "cert.pem", "key.pem", handler)
			} else {
				err = http.ListenAndServe(fmt.Sprintf(":%v", port), handler)
			}

			return err
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
