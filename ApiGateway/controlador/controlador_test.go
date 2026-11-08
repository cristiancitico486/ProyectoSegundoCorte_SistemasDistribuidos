package controlador

import (
	"bufio"
	"bytes"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gateway.local/api-gateway/descubrimiento"
)

// servicioFalso responde con una línea que describe lo que recibió, para
// comprobar a qué ruta del servicio llegó la petición del gateway.
func servicioFalso(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://otro.com") // el gateway debe descartarlo
		w.Header().Set("X-Servicio", "si")
		cuerpo, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusTeapot) // un código reconocible: debe pasar tal cual
		w.Write([]byte(r.Method + " " + r.URL.EscapedPath() + "?" + r.URL.RawQuery + " cuerpo=" + string(cuerpo)))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func nuevoGateway(t *testing.T, urls map[string]string) *httptest.Server {
	t.Helper()
	res, err := descubrimiento.NuevoEstatico(urls)
	if err != nil {
		t.Fatal(err)
	}
	gw := httptest.NewServer(NuevoControladorGateway(res).Manejador())
	t.Cleanup(gw.Close)
	return gw
}

func todosA(srv *httptest.Server) map[string]string {
	return map[string]string{
		descubrimiento.Metadata: srv.URL, descubrimiento.Audios: srv.URL,
		descubrimiento.Estadisticas: srv.URL, descubrimiento.Suscripciones: srv.URL,
		descubrimiento.Reacciones: srv.URL,
	}
}

func pedir(t *testing.T, metodo, url, cuerpo string, cab map[string]string) (*http.Response, string) {
	t.Helper()
	req, _ := http.NewRequest(metodo, url, strings.NewReader(cuerpo))
	for k, v := range cab {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	datos, _ := io.ReadAll(resp.Body)
	return resp, string(datos)
}

func TestTablaDeRutas(t *testing.T) {
	gw := nuevoGateway(t, todosA(servicioFalso(t)))
	casos := []struct{ metodo, ruta, cuerpo, esperado string }{
		{"GET", "/api/tipos", "", "GET /tipos? cuerpo="},
		{"GET", "/api/tipos/2/audios", "", "GET /tipos/2/audios? cuerpo="},
		{"GET", "/api/tipos/2/campos", "", "GET /tipos/2/campos? cuerpo="},
		{"GET", "/api/audios/9", "", "GET /audios/9? cuerpo="},
		{"POST", "/api/audios", `{"a":1}`, `POST /audios? cuerpo={"a":1}`},
		{"POST", "/api/audios/9/archivo", "datos", "POST /audios/9? cuerpo=datos"},
		{"GET", "/api/historial/u-ab12", "", "GET /estadisticas/historial/u-ab12? cuerpo="},
		{"GET", "/api/planes", "", "GET /planes? cuerpo="},
		{"POST", "/api/pagos", `{"token":"x"}`, `POST /pagos? cuerpo={"token":"x"}`},
		{"GET", "/api/suscripciones/u1", "", "GET /suscripciones/u1? cuerpo="},
		{"GET", "/ws/reacciones?usuarioId=u1", "", "GET /ws?usuarioId=u1 cuerpo="},
	}
	for _, c := range casos {
		resp, cuerpo := pedir(t, c.metodo, gw.URL+c.ruta, c.cuerpo, nil)
		if resp.StatusCode != http.StatusTeapot || cuerpo != c.esperado {
			t.Errorf("%s %s -> %d %q, esperado 418 %q", c.metodo, c.ruta, resp.StatusCode, cuerpo, c.esperado)
		}
		if resp.Header.Get("X-Servicio") != "si" {
			t.Errorf("%s: las cabeceras del servicio deben pasar", c.ruta)
		}
	}
}

func TestCORSUnicoYPreflight(t *testing.T) {
	gw := nuevoGateway(t, todosA(servicioFalso(t)))

	resp, _ := pedir(t, "GET", gw.URL+"/api/tipos", "", map[string]string{"Origin": "null"})
	if v := resp.Header.Values("Access-Control-Allow-Origin"); len(v) != 1 || v[0] != "*" {
		t.Fatalf("debe haber un único Allow-Origin '*' (el del servicio se descarta): %v", v)
	}

	resp, _ = pedir(t, "OPTIONS", gw.URL+"/api/pagos", "", map[string]string{
		"Origin": "null", "Access-Control-Request-Method": "POST",
		"Access-Control-Request-Headers": "content-type,x-raro",
	})
	if resp.StatusCode != 204 {
		t.Fatalf("preflight debe dar 204: %d", resp.StatusCode)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "*" ||
		!strings.Contains(resp.Header.Get("Access-Control-Allow-Methods"), "POST") ||
		!strings.Contains(resp.Header.Get("Access-Control-Allow-Headers"), "x-raro") ||
		!strings.Contains(resp.Header.Get("Access-Control-Allow-Headers"), "Idempotency-Key") {
		t.Fatalf("cabeceras de preflight incorrectas: %v", resp.Header)
	}
	// Preflight también en una ruta que no existe.
	if resp, _ = pedir(t, "OPTIONS", gw.URL+"/api/lo-que-sea", "", nil); resp.StatusCode != 204 {
		t.Fatalf("preflight en cualquier ruta: %d", resp.StatusCode)
	}
}

func TestErroresPropios(t *testing.T) {
	gw := nuevoGateway(t, todosA(servicioFalso(t)))
	resp, cuerpo := pedir(t, "GET", gw.URL+"/api/no-existe", "", nil)
	if resp.StatusCode != 404 || !strings.Contains(cuerpo, `"error":"ruta_no_encontrada"`) || resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("404: %d %s", resp.StatusCode, cuerpo)
	}
	resp, cuerpo = pedir(t, "GET", gw.URL+"/health", "", nil)
	if resp.StatusCode != 200 || !strings.Contains(cuerpo, `"status":"ok"`) {
		t.Fatalf("health: %d %s", resp.StatusCode, cuerpo)
	}

	// 503: servicio sin instancia (no configurado).
	solo := nuevoGateway(t, map[string]string{descubrimiento.Metadata: "http://127.0.0.1:1"})
	resp, cuerpo = pedir(t, "GET", solo.URL+"/api/planes", "", nil)
	if resp.StatusCode != 503 || !strings.Contains(cuerpo, `"error":"servicio_no_disponible"`) || !strings.Contains(cuerpo, `"mensaje"`) {
		t.Fatalf("503: %d %s", resp.StatusCode, cuerpo)
	}
	// 502: configurado pero no responde.
	resp, cuerpo = pedir(t, "GET", solo.URL+"/api/tipos", "", nil)
	if resp.StatusCode != 502 || !strings.Contains(cuerpo, `"error":"error_gateway"`) {
		t.Fatalf("502: %d %s", resp.StatusCode, cuerpo)
	}
}

func TestParametroConBarraNoCambiaLaRuta(t *testing.T) {
	gw := nuevoGateway(t, todosA(servicioFalso(t)))
	_, cuerpo := pedir(t, "GET", gw.URL+"/api/historial/a%2F..%2Fhealth", "", nil)
	if !strings.HasPrefix(cuerpo, "GET /estadisticas/historial/a%2F..%2Fhealth?") {
		t.Fatalf("el valor debe llegar como un único segmento codificado: %q", cuerpo)
	}
}

func TestLimitesDeCuerpo(t *testing.T) {
	gw := nuevoGateway(t, todosA(servicioFalso(t)))

	// Content-Length declarado demasiado grande: rechazo inmediato.
	req, _ := http.NewRequest("POST", gw.URL+"/api/pagos", strings.NewReader("x"))
	req.ContentLength = 2 << 20
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		defer resp.Body.Close()
		cuerpo, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != 413 || !strings.Contains(string(cuerpo), `"error":"cuerpo_muy_grande"`) {
			t.Fatalf("413: %d %s", resp.StatusCode, cuerpo)
		}
	}
	// Una subida de varios MB (mp3 de prueba) debe llegar completa.
	var cuerpoMultipart bytes.Buffer
	mp := multipart.NewWriter(&cuerpoMultipart)
	parte, _ := mp.CreateFormFile("archivo", "a.mp3")
	parte.Write(bytes.Repeat([]byte("ID3"), 2<<20)) // 6 MB
	mp.Close()
	tam := cuerpoMultipart.Len()
	resp2, texto := pedir(t, "POST", gw.URL+"/api/audios/9/archivo", cuerpoMultipart.String(), map[string]string{"Content-Type": mp.FormDataContentType()})
	if resp2.StatusCode != http.StatusTeapot || len(texto) < tam {
		t.Fatalf("la subida de 6 MB debe llegar completa: %d (%d de %d bytes)", resp2.StatusCode, len(texto), tam)
	}
}

// TestWebSocketTunel comprueba que el gateway deja pasar el Upgrade y
// conserva el query string; luego los bytes viajan en ambos sentidos.
func TestWebSocketTunel(t *testing.T) {
	recibido := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recibido <- r.URL.Path + "?" + r.URL.RawQuery + " origin=" + r.Header.Get("Origin")
		conn, buf, err := w.(http.Hijacker).Hijack()
		if err != nil {
			return
		}
		defer conn.Close()
		buf.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n")
		buf.Flush()
		linea, _ := buf.ReadString('\n') // eco de una línea
		buf.WriteString("eco:" + linea)
		buf.Flush()
	}))
	defer srv.Close()
	gw := nuevoGateway(t, todosA(srv))

	conn, err := net.DialTimeout("tcp", strings.TrimPrefix(gw.URL, "http://"), 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	io.WriteString(conn, "GET /ws/reacciones?usuarioId=u-ab12 HTTP/1.1\r\nHost: x\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n"+
		"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\nOrigin: null\r\n\r\n")
	lector := bufio.NewReader(conn)
	estado, _ := lector.ReadString('\n')
	if !strings.Contains(estado, "101") {
		t.Fatalf("debía responder 101: %q", estado)
	}
	for { // saltar cabeceras
		l, _ := lector.ReadString('\n')
		if l == "\r\n" || l == "" {
			break
		}
	}
	io.WriteString(conn, "hola\n")
	eco, _ := lector.ReadString('\n')
	if eco != "eco:hola\n" {
		t.Fatalf("el túnel debe llevar datos en ambos sentidos: %q", eco)
	}
	if v := <-recibido; v != "/ws?usuarioId=u-ab12 origin=null" {
		t.Fatalf("el servicio debe recibir /ws con el query string y el Origin: %q", v)
	}
}
