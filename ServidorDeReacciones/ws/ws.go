// Package ws implementa lo mínimo del protocolo WebSocket (RFC 6455) que
// necesita el servidor de reacciones: handshake, mensajes de texto, ping/pong
// y cierre. Usa solo la biblioteca estándar, así el módulo no añade
// dependencias externas.
package ws

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const guid = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

const (
	opContinuacion = 0x0
	opTexto        = 0x1
	opBinario      = 0x2
	opCierre       = 0x8
	opPing         = 0x9
	opPong         = 0xA
)

var (
	ErrMensajeGrande = errors.New("mensaje demasiado grande")
	ErrProtocolo     = errors.New("violación del protocolo WebSocket")
)

// Conn es una conexión WebSocket del lado servidor. Escribir es seguro desde
// varias goroutines; leer debe hacerse desde una sola.
type Conn struct {
	nc  net.Conn
	br  *bufio.Reader
	wmu sync.Mutex
	max int
}

// Aceptar valida el handshake, responde 101 y devuelve la conexión. Si la
// petición no es un handshake válido responde el error HTTP y devuelve error.
func Aceptar(w http.ResponseWriter, r *http.Request, maxMensaje int) (*Conn, error) {
	clave := r.Header.Get("Sec-WebSocket-Key")
	if r.Method != http.MethodGet || !contieneToken(r.Header.Get("Connection"), "upgrade") ||
		!strings.EqualFold(r.Header.Get("Upgrade"), "websocket") ||
		r.Header.Get("Sec-WebSocket-Version") != "13" || clave == "" {
		http.Error(w, "se esperaba un handshake WebSocket", http.StatusBadRequest)
		return nil, ErrProtocolo
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "el servidor no soporta WebSocket", http.StatusInternalServerError)
		return nil, errors.New("ResponseWriter sin Hijack")
	}
	nc, rw, err := hj.Hijack()
	if err != nil {
		return nil, err
	}
	suma := sha1.Sum([]byte(clave + guid))
	respuesta := "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + base64.StdEncoding.EncodeToString(suma[:]) + "\r\n\r\n"
	nc.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if _, err := nc.Write([]byte(respuesta)); err != nil {
		nc.Close()
		return nil, err
	}
	return &Conn{nc: nc, br: rw.Reader, max: maxMensaje}, nil
}

func contieneToken(cabecera, token string) bool {
	for _, t := range strings.Split(cabecera, ",") {
		if strings.EqualFold(strings.TrimSpace(t), token) {
			return true
		}
	}
	return false
}

// LeerMensaje bloquea hasta recibir un mensaje de texto completo. Responde
// solo a ping y cierre. plazo es el tiempo máximo de silencio permitido.
// Devuelve io.EOF cuando el cliente cierra.
func (c *Conn) LeerMensaje(plazo time.Duration) ([]byte, error) {
	var msg []byte
	fragmentando := false
	for {
		c.nc.SetReadDeadline(time.Now().Add(plazo))
		var h [2]byte
		if _, err := io.ReadFull(c.br, h[:]); err != nil {
			return nil, err
		}
		fin := h[0]&0x80 != 0
		opcode := h[0] & 0x0F
		if h[0]&0x70 != 0 || h[1]&0x80 == 0 { // bits RSV o cliente sin máscara
			return nil, ErrProtocolo
		}
		n := uint64(h[1] & 0x7F)
		switch n {
		case 126:
			var b [2]byte
			if _, err := io.ReadFull(c.br, b[:]); err != nil {
				return nil, err
			}
			n = uint64(binary.BigEndian.Uint16(b[:]))
		case 127:
			var b [8]byte
			if _, err := io.ReadFull(c.br, b[:]); err != nil {
				return nil, err
			}
			n = binary.BigEndian.Uint64(b[:])
		}
		if opcode >= opCierre && (n > 125 || !fin) {
			return nil, ErrProtocolo
		}
		if n > uint64(c.max) {
			return nil, ErrMensajeGrande
		}
		var mascara [4]byte
		if _, err := io.ReadFull(c.br, mascara[:]); err != nil {
			return nil, err
		}
		carga := make([]byte, n)
		if _, err := io.ReadFull(c.br, carga); err != nil {
			return nil, err
		}
		for i := range carga {
			carga[i] ^= mascara[i%4]
		}

		switch opcode {
		case opPing:
			c.escribir(opPong, carga)
			continue
		case opPong:
			continue
		case opCierre:
			c.escribir(opCierre, carga)
			return nil, io.EOF
		case opTexto:
			if fragmentando {
				return nil, ErrProtocolo
			}
			msg, fragmentando = carga, !fin
		case opContinuacion:
			if !fragmentando {
				return nil, ErrProtocolo
			}
			msg = append(msg, carga...)
			if len(msg) > c.max {
				return nil, ErrMensajeGrande
			}
			fragmentando = !fin
		default: // binario u opcode desconocido
			return nil, ErrProtocolo
		}
		if !fragmentando {
			return msg, nil
		}
	}
}

// EscribirTexto envía un mensaje de texto.
func (c *Conn) EscribirTexto(b []byte) error { return c.escribir(opTexto, b) }

// Ping envía un ping de control.
func (c *Conn) Ping() error { return c.escribir(opPing, nil) }

// Cerrar envía el cierre (1000) y libera la conexión.
func (c *Conn) Cerrar() {
	c.escribir(opCierre, []byte{0x03, 0xE8})
	c.nc.Close()
}

func (c *Conn) escribir(opcode byte, carga []byte) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	cab := []byte{0x80 | opcode}
	switch l := len(carga); {
	case l < 126:
		cab = append(cab, byte(l))
	case l <= 65535:
		cab = append(cab, 126, byte(l>>8), byte(l))
	default:
		cab = append(cab, 127)
		cab = binary.BigEndian.AppendUint64(cab, uint64(l))
	}
	c.nc.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err := c.nc.Write(append(cab, carga...))
	return err
}
