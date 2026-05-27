package core

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
)

const wsMagicGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

type wsConn struct {
	conn   net.Conn
	reader io.Reader
	mu     sync.Mutex
	closed bool
}

func wsUpgrade(w http.ResponseWriter, r *http.Request) (*wsConn, error) {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return nil, errors.New("not a websocket upgrade")
	}
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return nil, errors.New("missing key")
	}

	h := sha1.New()
	h.Write([]byte(key + wsMagicGUID))
	accept := base64.StdEncoding.EncodeToString(h.Sum(nil))

	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("hijack not supported")
	}
	conn, bufrw, err := hj.Hijack()
	if err != nil {
		return nil, err
	}

	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + accept + "\r\n\r\n"
	if _, err := bufrw.WriteString(resp); err != nil {
		conn.Close()
		return nil, err
	}
	if err := bufrw.Flush(); err != nil {
		conn.Close()
		return nil, err
	}
	return &wsConn{conn: conn, reader: bufrw.Reader}, nil
}

func (c *wsConn) writeText(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return errors.New("closed")
	}
	frame := []byte{0x81}
	if len(data) < 126 {
		frame = append(frame, byte(len(data)))
	} else if len(data) < 65536 {
		frame = append(frame, 126, byte(len(data)>>8), byte(len(data)))
	} else {
		frame = append(frame, 127)
		b := make([]byte, 8)
		for i := range 8 {
			b[i] = byte(len(data) >> (8 * (7 - i)))
		}
		frame = append(frame, b...)
	}
	frame = append(frame, data...)
	_, err := c.conn.Write(frame)
	return err
}

func (c *wsConn) readLoop(hub *eventHub, userID string) {
	defer func() { _ = c.conn.Close() }()
	hub.add(c, userID)
	defer hub.remove(c)
	for {
		hdr := make([]byte, 2)
		if _, err := io.ReadFull(c.reader, hdr); err != nil {
			return
		}
		op := hdr[0] & 0x0F
		masked := (hdr[1] & 0x80) != 0
		length := int64(hdr[1] & 0x7F)
		if length == 126 {
			b := make([]byte, 2)
			if _, err := io.ReadFull(c.reader, b); err != nil {
				return
			}
			length = int64(uint16(b[0])<<8 | uint16(b[1]))
		} else if length == 127 {
			b := make([]byte, 8)
			if _, err := io.ReadFull(c.reader, b); err != nil {
				return
			}
			length = 0
			for i := range 8 {
				length = (length << 8) | int64(b[i])
			}
		}
		if masked {
			mask := make([]byte, 4)
			if _, err := io.ReadFull(c.reader, mask); err != nil {
				return
			}
			_ = mask
		}
		if length > 0 {
			payload := make([]byte, length)
			if _, err := io.ReadFull(c.reader, payload); err != nil {
				return
			}
			if masked {
				for i := range payload {
					payload[i] ^= mask[i%4]
				}
			}
			_ = payload
		}
		if op == 8 {
			return
		}
		if op == 9 {
			_ = c.writeText([]byte{})
		}
	}
}

type eventHub struct {
	mu      sync.RWMutex
	clients map[*wsConn]string
}

func newEventHub() *eventHub {
	return &eventHub{clients: make(map[*wsConn]string)}
}

func (h *eventHub) add(c *wsConn, userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = userID
}

func (h *eventHub) remove(c *wsConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, c)
}

func (h *eventHub) broadcastAll(event interface{}) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		_ = c.writeText(data)
	}
}

func (h *eventHub) broadcastTo(userID string, event interface{}) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c, uid := range h.clients {
		if uid == userID {
			_ = c.writeText(data)
		}
	}
}
