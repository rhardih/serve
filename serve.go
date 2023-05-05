package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/NYTimes/gziphandler"
	"github.com/urfave/cli/v2"

	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
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

func customHeadersHandler(h http.Handler, headersMap map[string]string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for key, value := range headersMap {
			w.Header().Set(key, value)
		}
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

func createCertificateForKey(key *rsa.PrivateKey) ([]byte, error) {
	host := "localhost"
	validFor := 365 * 24 * time.Hour

	var err error

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

	return x509.CreateCertificate(rand.Reader, &template, &template, publicKey(key), key)
}

// This is extracted from src/crypto/tls/generate_cert.go, keeping only the
// bare minimum needed to create a usable cert for localhost development.
func generateOnDiskCert(path string) (string, string) {
	rsaBits := 2048

	var key *rsa.PrivateKey
	var err error
	key, err = rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		log.Fatalf("failed to generate private key: %s", err)
	}

	derBytes, err := createCertificateForKey(key)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}

	certPath := fmt.Sprintf("%s/%s", path, "cert.pem")
	keyPath := fmt.Sprintf("%s/%s", path, "key.pem")

	// Skip creating if the files already exists
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		_ = os.MkdirAll(path, os.ModePerm)

		certOut, err := os.Create(certPath)
		if err != nil {
			log.Fatalf("failed to open %s for writing: %s", certPath, err)
		}
		pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
		certOut.Close()
		log.Printf("written %s\n", certPath)

		keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			log.Print("failed to open %s for writing:", keyPath, err)
			return "", ""
		}

		pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
		keyOut.Close()
		log.Printf("written %s\n", keyPath)
	}

	return certPath, keyPath
}

func generateInMemoryCert() ([]byte, []byte, error) {
	// host := "localhost"
	// validFor := 365 * 24 * time.Hour
	rsaBits := 2048

	var key *rsa.PrivateKey
	var err error
	key, err = rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		log.Fatalf("failed to generate private key: %s", err)
	}

	derBytes, err := createCertificateForKey(key)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}

	certArray := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyArray := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	return certArray, keyArray, nil
}

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	var gzip bool
	var port int
	var path string = "./"
	var logging bool
	var http2 bool
	var headerFlags cli.StringSlice
	var headersMap map[string]string
	var certSave bool
	var certDir string = fmt.Sprintf("%s/.serve", homeDir)
	var stop = make(chan os.Signal, 1)
	var server *http.Server

	app := &cli.App{
		Name:    "serve",
		Usage:   "deliver content of current directory via http/https",
		Version: "2.0.0",

		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "gzip",
				Aliases:     []string{"g"},
				Usage:       "enable gzip encoding",
				Destination: &gzip,
			},
			&cli.IntFlag{
				Name:        "port",
				Aliases:     []string{"p"},
				Usage:       "specify port for listening",
				Value:       8080,
				Destination: &port,
			},
			&cli.BoolFlag{
				Name:        "logging",
				Aliases:     []string{"l"},
				Usage:       "enable logging output",
				Destination: &logging,
			},
			&cli.BoolFlag{
				Name:        "http2",
				Aliases:     []string{"2"},
				Usage:       "enable http2, this generates a self signed certificate, if one isn't already present; cert.pem, key.pem",
				Destination: &http2,
			},
			&cli.StringSliceFlag{
				Name:        "header",
				Aliases:     []string{"H"},
				Usage:       "custom header(s) to add to the response (can be repeated multiple times)",
				Destination: &headerFlags,
			},
			&cli.BoolFlag{
				Name:        "cert-save",
				Aliases:     []string{"sc"},
				Usage:       "whether to save the generated self-signed certificates to disk",
				Destination: &certSave,
			},
			&cli.StringFlag{
				Name:        "cert-dir",
				Aliases:     []string{"cd"},
				Usage:       "location to save certificate at if saving to disk",
				Value:       certDir,
				Destination: &certDir,
			},
		},

		Action: func(c *cli.Context) error {
			if c.NArg() > 0 {
				path = c.Args().Get(0)
			}

			handler := http.FileServer(http.Dir(path))

			if gzip {
				handler = gziphandler.GzipHandler(handler)
			}

			if logging {
				handler = logHandler(handler)
			}

			headers := headerFlags.Value()
			if len(headers) > 0 {
				headersMap = make(map[string]string)
				for _, header := range headers {
					idx := strings.Index(header, ":")
					if idx < 0 {
						return fmt.Errorf("Invalid header: %s", header)
					}
					k := header[:idx]
					v := header[idx+1:]
					headersMap[strings.TrimSpace(k)] = strings.TrimSpace(v)
				}
				handler = customHeadersHandler(handler, headersMap)
			}

			server = &http.Server{
				Addr:    fmt.Sprintf(":%d", port),
				Handler: handler,
			}

			serveHttps := func() {
				if certSave {
					certPath, keyPath := generateOnDiskCert(certDir)

					err = server.ListenAndServeTLS(certPath, keyPath)
				} else {
					cert, key, err := generateInMemoryCert()
					if err != nil {
						log.Fatal("Error: Couldn't create https certs.")
					}

					keyPair, err := tls.X509KeyPair(cert, key)
					if err != nil {
						log.Fatal(err)
						log.Fatal("Error: Couldn't create key pair")
					}

					var certificates []tls.Certificate
					certificates = append(certificates, keyPair)

					cfg := &tls.Config{
						MinVersion:               tls.VersionTLS12,
						PreferServerCipherSuites: true,
						Certificates:             certificates,
					}

					server.TLSConfig = cfg

					log.Fatal(server.ListenAndServeTLS("", ""))
				}

				if err != nil {
					log.Fatalf("ListenAndServeTLS error: %s", err)
				}
			}

			serveHttp := func() {
				err = server.ListenAndServe()
				if err != nil {
					log.Fatalf("ListenAndServe error: %s", err)
				}
			}

			if http2 {
				go serveHttps()
			} else {
				go serveHttp()
			}

			log.Println(fmt.Sprintf("Serving content of %s on localhost:%v ...", path, port))

			signal.Notify(stop, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

			<-stop

			log.Print("Stopping.")

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := server.Shutdown(ctx); err != nil {
				log.Fatalf("%s", err)
			}

			return nil
		},
	}

	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
