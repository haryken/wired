package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"time"
)

const (
	wiredTLSDir  = "/data/wired/tls"
	wiredTLSCert = "/data/wired/tls/cert.pem"
	wiredTLSKey  = "/data/wired/tls/key.pem"
	wiredHTTPS   = ":8443"
)

// ensureWiredTLS returns cert/key paths, generating a long-lived self-signed
// cert if needed so browsers treat the UI as a secure context (required for mic).
func ensureWiredTLS() (certFile, keyFile string, err error) {
	if _, err1 := os.Stat(wiredTLSCert); err1 == nil {
		if _, err2 := os.Stat(wiredTLSKey); err2 == nil {
			return wiredTLSCert, wiredTLSKey, nil
		}
	}
	if err := os.MkdirAll(wiredTLSDir, 0755); err != nil {
		return "", "", err
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", "", err
	}
	host, _ := os.Hostname()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "wired", Organization: []string{"WireOS"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost", "wired.local"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}
	if host != "" {
		tmpl.DNSNames = append(tmpl.DNSNames, host)
	}
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			if ip4 := ip.To4(); ip4 != nil {
				tmpl.IPAddresses = append(tmpl.IPAddresses, ip4)
			}
		}
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return "", "", err
	}
	certOut, err := os.OpenFile(wiredTLSCert, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return "", "", err
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		certOut.Close()
		return "", "", err
	}
	certOut.Close()

	keyOut, err := os.OpenFile(wiredTLSKey, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", "", err
	}
	b, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		keyOut.Close()
		return "", "", err
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}); err != nil {
		keyOut.Close()
		return "", "", err
	}
	keyOut.Close()
	return wiredTLSCert, wiredTLSKey, nil
}

func startHTTPS(handler http.Handler) {
	cert, key, err := ensureWiredTLS()
	if err != nil {
		fmt.Println("wired TLS cert:", err)
		return
	}
	fmt.Println("starting HTTPS web at", wiredHTTPS, "(use this URL for microphone)")
	go func() {
		if err := http.ListenAndServeTLS(wiredHTTPS, cert, key, handler); err != nil {
			fmt.Println("wired listen", wiredHTTPS, "failed:", err)
		}
	}()
}
