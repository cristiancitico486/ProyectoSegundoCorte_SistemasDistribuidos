package consul

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRegistrarYDesregistrar(t *testing.T) {
	recibido := make(chan registro, 1)
	desreg := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/agent/service/register":
			var reg registro
			b, _ := io.ReadAll(r.Body)
			json.Unmarshal(b, &reg)
			recibido <- reg
		default:
			desreg <- r.URL.Path
		}
	}))
	defer srv.Close()

	baja := registrarEn(srv.URL, "demo-service", 8099, "/health")
	select {
	case reg := <-recibido:
		if reg.Name != "demo-service" || reg.Port != 8099 || reg.Check.HTTP != "http://127.0.0.1:8099/health" {
			t.Fatalf("registro inesperado: %+v", reg)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no se registró")
	}
	baja()
	if p := <-desreg; p != "/v1/agent/service/deregister/demo-service-8099" {
		t.Fatalf("ruta de baja inesperada: %s", p)
	}
}

func TestSinConsulAddrNoHaceNada(t *testing.T) {
	t.Setenv("CONSUL_ADDR", "")
	Registrar("x", 1, "")()
}
