// Package descubrimiento resuelve a qué dirección enviar cada petición del
// gateway. Con CONSUL_ADDR consulta a Consul las instancias sanas de cada
// servicio (con caché corta y reparto round-robin); sin Consul usa URLs fijas.
package descubrimiento

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Nombres de los servicios en Consul (contratos v0.3, sección 0).
const (
	Metadata      = "metadata-service"
	Audios        = "audios-service"
	Reacciones    = "reacciones-service"
	Estadisticas  = "estadisticas-service"
	Suscripciones = "suscripciones-service"

	ttlCacheConsul = 5 * time.Second
)

// ErrSinInstancias indica que no hay ninguna instancia sana del servicio.
var ErrSinInstancias = errors.New("no hay instancias sanas del servicio")

// Resolutor entrega la URL base de una instancia del servicio pedido.
type Resolutor interface {
	Resolver(servicio string) (*url.URL, error)
}

// ---------------------------------------------------------------- Estático

// Estatico resuelve con URLs fijas (modo sin Consul).
type Estatico struct {
	urls map[string]*url.URL
}

// NuevoEstatico valida y guarda las URLs base por nombre de servicio.
func NuevoEstatico(urls map[string]string) (*Estatico, error) {
	r := &Estatico{urls: make(map[string]*url.URL, len(urls))}
	for servicio, crudo := range urls {
		u, err := url.Parse(crudo)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return nil, fmt.Errorf("URL inválida para %s: %q", servicio, crudo)
		}
		r.urls[servicio] = u
	}
	return r, nil
}

func (r *Estatico) Resolver(servicio string) (*url.URL, error) {
	u, ok := r.urls[servicio]
	if !ok {
		return nil, ErrSinInstancias
	}
	copia := *u
	return &copia, nil
}

// ------------------------------------------------------------------ Consul

type entrada struct {
	instancias []string // "host:puerto"
	vence      time.Time
	siguiente  int
}

func (e *entrada) siguienteURL() (*url.URL, error) {
	if len(e.instancias) == 0 {
		return nil, ErrSinInstancias
	}
	host := e.instancias[e.siguiente%len(e.instancias)]
	e.siguiente++
	return &url.URL{Scheme: "http", Host: host}, nil
}

// Consul resuelve consultando /v1/health/service/<nombre>?passing.
type Consul struct {
	base    string
	cliente *http.Client
	ttl     time.Duration
	ahora   func() time.Time

	mu    sync.Mutex
	cache map[string]*entrada
}

// NuevoConsul acepta la dirección con o sin esquema ("localhost:8500" o
// "http://localhost:8500").
func NuevoConsul(direccion string) *Consul {
	base := strings.TrimRight(direccion, "/")
	if !strings.Contains(base, "://") {
		base = "http://" + base
	}
	return &Consul{
		base:    base,
		cliente: &http.Client{Timeout: 3 * time.Second},
		ttl:     ttlCacheConsul,
		ahora:   time.Now,
		cache:   map[string]*entrada{},
	}
}

func (c *Consul) Resolver(servicio string) (*url.URL, error) {
	c.mu.Lock()
	if e := c.cache[servicio]; e != nil && c.ahora().Before(e.vence) {
		defer c.mu.Unlock()
		return e.siguienteURL()
	}
	c.mu.Unlock()

	instancias, err := c.consultar(servicio)

	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.cache[servicio]
	if err != nil {
		// Consul no responde: se sigue con la última respuesta conocida.
		if e != nil && len(e.instancias) > 0 {
			fmt.Printf("[Descubrimiento] Consul no responde (%v): se usa la última respuesta conocida de %s\n", err, servicio)
			e.vence = c.ahora().Add(c.ttl)
			return e.siguienteURL()
		}
		return nil, fmt.Errorf("no se pudo consultar Consul: %w", err)
	}
	if e == nil {
		e = &entrada{}
		c.cache[servicio] = e
	}
	if strings.Join(e.instancias, ",") != strings.Join(instancias, ",") {
		fmt.Printf("[Descubrimiento] %s: %d instancia(s) sana(s) %v\n", servicio, len(instancias), instancias)
	}
	e.instancias = instancias
	e.vence = c.ahora().Add(c.ttl)
	return e.siguienteURL()
}

type respuestaSalud struct {
	Node struct {
		Address string
	}
	Service struct {
		Address string
		Port    int
	}
}

func (c *Consul) consultar(servicio string) ([]string, error) {
	resp, err := c.cliente.Get(c.base + "/v1/health/service/" + url.PathEscape(servicio) + "?passing")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Consul respondió %d", resp.StatusCode)
	}
	var entradas []respuestaSalud
	if err := json.NewDecoder(resp.Body).Decode(&entradas); err != nil {
		return nil, fmt.Errorf("respuesta de Consul ilegible: %w", err)
	}
	instancias := make([]string, 0, len(entradas))
	for _, e := range entradas {
		host := e.Service.Address
		if host == "" {
			host = e.Node.Address
		}
		if host == "" || e.Service.Port == 0 {
			continue
		}
		instancias = append(instancias, net.JoinHostPort(host, strconv.Itoa(e.Service.Port)))
	}
	return instancias, nil
}
