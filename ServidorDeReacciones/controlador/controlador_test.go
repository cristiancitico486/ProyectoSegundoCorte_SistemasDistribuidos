package controlador

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"reacciones.local/servidor-reacciones/fachada"
	"reacciones.local/servidor-reacciones/repositorio"
)

// cliente mínimo de WebSocket para las pruebas (frames con máscara).
type cli struct {
	t  *testing.T
	nc net.Conn
	br *bufio.Reader
}

func nuevoServidor(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()
	NuevoControladorReacciones(fachada.NuevaFachadaReacciones(repositorio.NuevoRepositorioRegistro())).RegistrarRutas(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func conectar(t *testing.T, srv *httptest.Server, usuario string) *cli {
	nc, err := net.Dial("tcp", strings.TrimPrefix(srv.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { nc.Close() })
	req := "GET /ws?usuarioId=" + usuario + " HTTP/1.1\r\nHost: x\r\nUpgrade: websocket\r\nConnection: keep-alive, Upgrade\r\n" +
		"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\n\r\n"
	nc.Write([]byte(req))
	br := bufio.NewReader(nc)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 101 || resp.Header.Get("Sec-WebSocket-Accept") != "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=" {
		t.Fatalf("handshake inválido: %d %v", resp.StatusCode, resp.Header)
	}
	return &cli{t: t, nc: nc, br: br}
}

func (c *cli) enviar(texto string) {
	mask := [4]byte{1, 2, 3, 4}
	var cab []byte
	switch l := len(texto); {
	case l < 126:
		cab = []byte{0x81, 0x80 | byte(l)}
	default:
		cab = append([]byte{0x81, 0x80 | 126}, 0, 0)
		binary.BigEndian.PutUint16(cab[2:], uint16(l))
	}
	carga := []byte(texto)
	for i := range carga {
		carga[i] ^= mask[i%4]
	}
	c.nc.Write(append(append(cab, mask[:]...), carga...))
}

// leer devuelve el siguiente mensaje JSON, o nil si no llega nada en 300 ms.
func (c *cli) leer() map[string]any {
	c.nc.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	var h [2]byte
	if _, err := io.ReadFull(c.br, h[:]); err != nil {
		return nil
	}
	n := int(h[1] & 0x7F)
	if n == 126 {
		var b [2]byte
		io.ReadFull(c.br, b[:])
		n = int(binary.BigEndian.Uint16(b[:]))
	}
	carga := make([]byte, n)
	io.ReadFull(c.br, carga)
	var m map[string]any
	if err := json.Unmarshal(carga, &m); err != nil {
		c.t.Fatalf("JSON inválido del servidor: %s", carga)
	}
	return m
}

func (c *cli) esperar(campo, valor string) map[string]any {
	m := c.leer()
	if m == nil || m[campo] != valor {
		c.t.Fatalf("se esperaba %s=%s y llegó %v", campo, valor, m)
	}
	return m
}

func TestReaccionLlegaSoloALosDelMismoAudio(t *testing.T) {
	srv := nuevoServidor(t)
	a, b, c := conectar(t, srv, "ana"), conectar(t, srv, "beto"), conectar(t, srv, "caro")
	a.enviar(`{"tipo":"registrar","audioId":5}`)
	a.esperar("tipo", "ack")
	b.enviar(`{"tipo":"registrar","audioId":5}`)
	if m := b.esperar("tipo", "ack"); m["oyentes"] != float64(2) {
		t.Fatalf("oyentes esperados 2: %v", m)
	}
	c.enviar(`{"tipo":"registrar","audioId":6}`)
	c.esperar("tipo", "ack")

	a.enviar(`{"tipo":"reaccion","audioId":5,"reaccion":"me_gusta"}`)
	for _, x := range []*cli{a, b} { // incluido el emisor
		m := x.esperar("tipo", "reaccion")
		if m["usuarioId"] != "ana" || m["reaccion"] != "me_gusta" || m["audioId"] != float64(5) || m["timestamp"] == "" {
			t.Fatalf("callback inesperado: %v", m)
		}
	}
	if m := c.leer(); m != nil {
		t.Fatalf("caro (otro audio) no debía recibir nada: %v", m)
	}
}

func TestErrores(t *testing.T) {
	srv := nuevoServidor(t)
	a := conectar(t, srv, "ana")
	a.enviar(`{"tipo":"reaccion","audioId":5,"reaccion":"fuego"}`)
	a.esperar("codigo", "no_registrado")
	a.enviar(`{"tipo":"registrar","audioId":5}`)
	a.esperar("tipo", "ack")
	a.enviar(`{"tipo":"reaccion","audioId":5,"reaccion":"enojo"}`)
	a.esperar("codigo", "reaccion_invalida")
	a.enviar(`no es json`)
	a.esperar("codigo", "mensaje_invalido")
	a.enviar(`{"tipo":"registrar"}`)
	a.esperar("codigo", "mensaje_invalido")
	a.enviar(`{"tipo":"bailar","audioId":1}`)
	a.esperar("codigo", "mensaje_invalido")
	a.enviar(`{"tipo":"reaccion","audioId":9,"reaccion":"fuego"}`) // registrado en otro audio
	a.esperar("codigo", "no_registrado")
}

func TestDesregistrarYCambioDeAudio(t *testing.T) {
	srv := nuevoServidor(t)
	a, b := conectar(t, srv, "ana"), conectar(t, srv, "beto")
	a.enviar(`{"tipo":"registrar","audioId":5}`)
	a.esperar("tipo", "ack")
	b.enviar(`{"tipo":"registrar","audioId":5}`)
	b.esperar("tipo", "ack")

	b.enviar(`{"tipo":"desregistrar","audioId":5}`)
	if m := b.esperar("operacion", "desregistrar"); m["oyentes"] != float64(1) {
		t.Fatalf("oyentes esperados 1: %v", m)
	}
	a.enviar(`{"tipo":"reaccion","audioId":5,"reaccion":"risa"}`)
	a.esperar("tipo", "reaccion")
	if m := b.leer(); m != nil {
		t.Fatalf("beto desregistrado no debía recibir: %v", m)
	}

	// registrar en otro audio saca a ana del 5
	a.enviar(`{"tipo":"registrar","audioId":7}`)
	a.esperar("tipo", "ack")
	b.enviar(`{"tipo":"registrar","audioId":5}`)
	if m := b.esperar("tipo", "ack"); m["oyentes"] != float64(1) {
		t.Fatalf("ana ya no debía estar en el 5: %v", m)
	}
}

func TestDesconexionLiberaRegistro(t *testing.T) {
	srv := nuevoServidor(t)
	a, b := conectar(t, srv, "ana"), conectar(t, srv, "beto")
	a.enviar(`{"tipo":"registrar","audioId":5}`)
	a.esperar("tipo", "ack")
	a.nc.Close()
	time.Sleep(100 * time.Millisecond)
	b.enviar(`{"tipo":"registrar","audioId":5}`)
	if m := b.esperar("tipo", "ack"); m["oyentes"] != float64(1) {
		t.Fatalf("ana desconectada no debía contar: %v", m)
	}
}

func TestSinUsuarioIdYHealth(t *testing.T) {
	srv := nuevoServidor(t)
	resp, err := http.Get(srv.URL + "/ws")
	if err != nil || resp.StatusCode != 400 {
		t.Fatalf("se esperaba 400 sin usuarioId: %v %v", resp, err)
	}
	resp, _ = http.Get(srv.URL + "/health")
	var m map[string]string
	json.NewDecoder(resp.Body).Decode(&m)
	if m["status"] != "ok" {
		t.Fatalf("health: %v", m)
	}
}

func TestMensajeMuyGrandeCierra(t *testing.T) {
	srv := nuevoServidor(t)
	a := conectar(t, srv, "ana")
	a.enviar(`{"tipo":"registrar","audioId":5,"reaccion":"` + strings.Repeat("x", 1100) + `"}`)
	a.esperar("codigo", "mensaje_invalido")
}
