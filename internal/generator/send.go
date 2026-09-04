package generator

import (
	"bytes"
	"net"
	"net/http"
	"strconv"
)

// sendUDPOnce dials and writes a single UDP datagram, used by replay
// (which sends pre-recorded payloads rather than the continuous-connection
// pattern the junk generators use).
func sendUDPOnce(host string, port int, payload []byte) (int, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("udp", addr, tcpDialTimeout)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	return conn.Write(payload)
}

// sendTCPOnce connects, writes payload, and closes.
func sendTCPOnce(host string, port int, payload []byte) (int, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, tcpDialTimeout)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	return conn.Write(payload)
}

// sendHTTPOnce POSTs payload to host:port with the standard thugsflooder
// markers, for replay of http-flavored recording entries.
func sendHTTPOnce(client *http.Client, host string, port int, payload []byte, session string) (int, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	url := "http://" + addr + "/thugsflooder-replay/" + session

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "thugsflooder-synthetic-test-traffic")
	req.Header.Set("X-Thugsflooder", Marker+"|session="+session)

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return len(payload), nil
}
