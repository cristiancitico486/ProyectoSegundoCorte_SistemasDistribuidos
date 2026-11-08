package descubrimiento

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// consulFalso devuelve el JSON de /v1/health/service/<nombre>?passing.
func consulFalso(t *testing.T, cuerpo string, llamadas *int32, estado *int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(llamadas, 1)
		if r.URL.Path != "/v1/health/service/metadata-service" || r.URL.RawQuery != "passing" {
			t.Errorf("consulta inesperada: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		if s := atomic.LoadInt32(estado); s != 0 {
			w.WriteHeader(int(s))
			return
		}
		w.Write([]byte(cuerpo))
	}))
	t.Cleanup(srv.Close)
	return srv
}

const dosInstancias = `[
 {"Node":{"Address":"10.0.0.1"},"Service":{"Address":"127.0.0.1","Port":8081}},
 {"Node":{"Address":"10.0.0.2"},"Service":{"Address":"","Port":8081}}]`

func TestConsulRoundRobinYDireccionDeNodo(t *testing.T) {
	var llamadas, estado int32
	srv := consulFalso(t, dosInstancias, &llamadas, &estado)
	c := NuevoConsul(srv.URL)

	var hosts []string
	for i := 0; i < 4; i++ {
		u, err := c.Resolver(Metadata)
		if err != nil {
			t.Fatal(err)
		}
		hosts = append(hosts, u.Host)
	}
	esperado := []string{"127.0.0.1:8081", "10.0.0.2:8081", "127.0.0.1:8081", "10.0.0.2:8081"}
	for i := range esperado {
		if hosts[i] != esperado[i] {
			t.Fatalf("round-robin %v, esperado %v", hosts, esperado)
		}
	}
	if llamadas != 1 {
		t.Fatalf("la caché debe evitar consultas repetidas: %d llamadas", llamadas)
	}
}

func TestConsulCacheVenceYSeRefresca(t *testing.T) {
	var llamadas, estado int32
	srv := consulFalso(t, dosInstancias, &llamadas, &estado)
	c := NuevoConsul(srv.URL)
	ahora := time.Now()
	c.ahora = func() time.Time { return ahora }

	c.Resolver(Metadata)
	ahora = ahora.Add(4 * time.Second)
	c.Resolver(Metadata)
	if llamadas != 1 {
		t.Fatalf("a los 4 s aún debe usar la caché: %d", llamadas)
	}
	ahora = ahora.Add(2 * time.Second)
	c.Resolver(Metadata)
	if llamadas != 2 {
		t.Fatalf("a los 6 s debe consultar de nuevo: %d", llamadas)
	}
}

func TestConsulUsaUltimaRespuestaSiFalla(t *testing.T) {
	var llamadas, estado int32
	srv := consulFalso(t, dosInstancias, &llamadas, &estado)
	c := NuevoConsul(srv.URL)
	ahora := time.Now()
	c.ahora = func() time.Time { return ahora }

	if _, err := c.Resolver(Metadata); err != nil {
		t.Fatal(err)
	}
	atomic.StoreInt32(&estado, 500) // Consul empieza a fallar
	ahora = ahora.Add(10 * time.Second)
	if _, err := c.Resolver(Metadata); err != nil {
		t.Fatalf("debía usar la última respuesta conocida: %v", err)
	}
}

func TestConsulSinInstanciasYConsultaFallida(t *testing.T) {
	var llamadas, estado int32
	srv := consulFalso(t, `[]`, &llamadas, &estado)
	c := NuevoConsul(srv.URL)
	if _, err := c.Resolver(Metadata); !errors.Is(err, ErrSinInstancias) {
		t.Fatalf("sin instancias sanas debe dar ErrSinInstancias: %v", err)
	}

	// Consul caído y sin caché previa.
	caido := NuevoConsul("127.0.0.1:1")
	if _, err := caido.Resolver(Metadata); err == nil || errors.Is(err, ErrSinInstancias) {
		t.Fatalf("Consul caído debe dar error de consulta: %v", err)
	}
}

func TestEstatico(t *testing.T) {
	r, err := NuevoEstatico(map[string]string{Metadata: "http://localhost:8081"})
	if err != nil {
		t.Fatal(err)
	}
	if u, err := r.Resolver(Metadata); err != nil || u.Host != "localhost:8081" {
		t.Fatalf("resolver estático: %v %v", u, err)
	}
	if _, err := r.Resolver(Audios); !errors.Is(err, ErrSinInstancias) {
		t.Fatalf("servicio no configurado: %v", err)
	}
	if _, err := NuevoEstatico(map[string]string{Metadata: "localhost:8081"}); err == nil {
		t.Fatalf("una URL sin esquema debe rechazarse")
	}
}
