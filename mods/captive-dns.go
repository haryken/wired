package mods

import (
	"encoding/binary"
	"net"
	"sync"
	"time"
)

// Captive DNS: answer every A query with the open-AP IP so phones' OS
// connectivity checks (captive.apple.com, generate_204 hosts, …) hit wired:80
// and auto-open the portal — same idea as Xiaozhi / esp-wifi-connect DnsServer.
//
// If dnsmasq already owns :53 (vic-setup-ap preferred path), bind fails and
// we rely on dnsmasq --address=/#/192.168.4.1 instead.

var (
	captiveDNSMu   sync.Mutex
	captiveDNSConn net.PacketConn
	captiveDNSStop chan struct{}
)

func ensureCaptiveDNS() {
	captiveDNSMu.Lock()
	defer captiveDNSMu.Unlock()
	if captiveDNSConn != nil {
		return
	}
	conn, err := net.ListenPacket("udp", "0.0.0.0:53")
	if err != nil {
		wifiLog("captive DNS :53 busy/unavailable (%v) — use dnsmasq if present", err)
		return
	}
	stop := make(chan struct{})
	captiveDNSConn = conn
	captiveDNSStop = stop
	go runCaptiveDNS(conn, stop)
	wifiLog("captive DNS hijack listening on :53 → %s", wifiOpenAPIP)
}

func stopCaptiveDNS() {
	captiveDNSMu.Lock()
	defer captiveDNSMu.Unlock()
	if captiveDNSStop != nil {
		select {
		case <-captiveDNSStop:
		default:
			close(captiveDNSStop)
		}
		captiveDNSStop = nil
	}
	if captiveDNSConn != nil {
		_ = captiveDNSConn.Close()
		captiveDNSConn = nil
		wifiLog("captive DNS stopped")
	}
}

func captiveDNSWatch() {
	for {
		if tetheringOn() {
			ensureCaptiveDNS()
		} else {
			stopCaptiveDNS()
		}
		time.Sleep(2 * time.Second)
	}
}

func runCaptiveDNS(conn net.PacketConn, stop <-chan struct{}) {
	buf := make([]byte, 512)
	ip := net.ParseIP(wifiOpenAPIP).To4()
	if ip == nil {
		return
	}
	for {
		select {
		case <-stop:
			return
		default:
		}
		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			select {
			case <-stop:
				return
			default:
				continue
			}
		}
		if n < 12 {
			continue
		}
		resp := buildCaptiveDNSResponse(buf[:n], ip)
		if len(resp) == 0 {
			continue
		}
		_, _ = conn.WriteTo(resp, addr)
	}
}

// buildCaptiveDNSResponse turns a query into a single A answer for wifiOpenAPIP.
// Handles standard queries; skips malformed packets.
func buildCaptiveDNSResponse(req []byte, ip net.IP) []byte {
	if len(req) < 12 {
		return nil
	}
	// QDCOUNT
	qd := binary.BigEndian.Uint16(req[4:6])
	if qd == 0 {
		return nil
	}
	// Skip question section; remember first QTYPE.
	i := 12
	var qtype uint16
	for q := 0; q < int(qd); q++ {
		for i < len(req) {
			l := int(req[i])
			i++
			if l == 0 {
				break
			}
			if l >= 0xc0 {
				if i >= len(req) {
					return nil
				}
				i++
				break
			}
			i += l
			if i > len(req) {
				return nil
			}
		}
		if i+4 > len(req) {
			return nil
		}
		if q == 0 {
			qtype = binary.BigEndian.Uint16(req[i : i+2])
		}
		i += 4
	}
	qEnd := i

	out := make([]byte, 0, qEnd+16)
	out = append(out, req[:qEnd]...)
	// Flags: QR | AA | RA; keep RD from request
	out[2] = 0x84
	if req[2]&0x01 != 0 {
		out[2] |= 0x01
	}
	out[3] = 0x80
	binary.BigEndian.PutUint16(out[8:10], 0)  // NSCOUNT
	binary.BigEndian.PutUint16(out[10:12], 0) // ARCOUNT

	// A → point everything at the AP. AAAA → empty success (force IPv4 path).
	if qtype == 1 { // A
		binary.BigEndian.PutUint16(out[6:8], 1) // ANCOUNT
		out = append(out,
			0xc0, 0x0c,
			0x00, 0x01,
			0x00, 0x01,
			0x00, 0x00, 0x00, 0x3c,
			0x00, 0x04,
			ip[0], ip[1], ip[2], ip[3],
		)
	} else {
		binary.BigEndian.PutUint16(out[6:8], 0) // ANCOUNT
	}
	return out
}
