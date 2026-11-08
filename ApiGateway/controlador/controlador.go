// Package controlador es la capa HTTP del gateway: registra las rutas
// públicas, aplica CORS y reenvía cada petición al servicio que corresponde.
// El cuerpo y el código de estado del servicio destino pasan sin cambios; el
// gateway solo responde por su cuenta cuando no puede llegar al servicio.
package controlador

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"time"

	"gateway.local/api-gateway/descubrimiento"
	"gateway.local/api-gateway/dto"
)

type ControladorGateway struct {
	resolutor  descubrimiento.Resolutor
	transporte http.RoundTripper
}

func NuevoControladorGateway(r descubrimiento.Resolutor) *ControladorGateway {
	return &ControladorGateway{
		resolutor: r,
		// Transporte compartido (reutiliza conexiones). No usa proxies del
		// entorno y no pone límite al cuerpo: la subida de un mp3 y los
		// WebSocket pueden durar minutos.
		transporte: &http.Transport{
			Proxy: nil,
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          50,
			IdleConnTimeout:       90 * time.Second,
			ResponseHeaderTimeout: 60 * time.Second,
		},
	}
}

// Manejador devuelve el handler completo (rutas + CORS) para el http.Server.
func (c *ControladorGateway) Manejador() http.Handler {
	mux := http.NewServeMux()
	for _, rt := range rutas {
		mux.HandleFunc(rt.Metodo+" "+rt.Patron, c.reenviar(rt))
	}
	mux.HandleFunc("GET /health", c.Salud)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		responderError(w, http.StatusNotFound, "ruta_no_encontrada", "la ruta "+r.Method+" "+r.URL.Path+" no existe en el gateway")
	})
	return ConCORS(mux)
}

// Salud atiende GET /health: lo responde el propio gateway.
func (c *ControladorGateway) Salud(w http.ResponseWriter, r *http.Request) {
	responderJSON(w, http.StatusOK, dto.SaludDTO{Status: "ok"})
}

func responderJSON(w http.ResponseWriter, status int, cuerpo any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(cuerpo)
}

func responderError(w http.ResponseWriter, status int, codigo, mensaje string) {
	responderJSON(w, status, dto.ErrorDTO{Error: codigo, Mensaje: mensaje})
}

var marcador = regexp.MustCompile(`\{(\w+)\}`)

// construirDestino sustituye los {parámetros} de la ruta del servicio por los
// valores de la petición. Devuelve la ruta decodificada y la codificada, para
// que un valor con "/" o "%2F" no pueda cambiar la ruta que llega al servicio.
func construirDestino(plantilla string, r *http.Request) (path, rawPath string) {
	path = marcador.ReplaceAllStringFunc(plantilla, func(m string) string {
		return r.PathValue(m[1 : len(m)-1])
	})
	rawPath = marcador.ReplaceAllStringFunc(plantilla, func(m string) string {
		return url.PathEscape(r.PathValue(m[1 : len(m)-1]))
	})
	return path, rawPath
}

func esWebSocket(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

// reenviar crea el handler que reenvía una ruta a su servicio.
func (c *ControladorGateway) reenviar(rt ruta) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if rt.MaxCuerpo > 0 {
			if r.ContentLength > rt.MaxCuerpo {
				responderError(w, http.StatusRequestEntityTooLarge, "cuerpo_muy_grande", "el cuerpo de la petición supera el tamaño máximo permitido")
				return
			}
			// Cubre las peticiones sin Content-Length (chunked).
			r.Body = http.MaxBytesReader(w, r.Body, rt.MaxCuerpo)
		}

		destino, err := c.resolutor.Resolver(rt.Servicio)
		if err != nil {
			fmt.Printf("[Gateway] %s %s -> %s: sin instancia disponible (%v)\n", r.Method, r.URL.Path, rt.Servicio, err)
			responderError(w, http.StatusServiceUnavailable, "servicio_no_disponible",
				"no hay una instancia disponible de "+rt.Servicio)
			return
		}

		etiqueta := r.Method
		if esWebSocket(r) {
			etiqueta = "WS"
		}
		fmt.Printf("[Gateway] %s %s -> %s (%s)\n", etiqueta, r.URL.Path, rt.Servicio, destino.Host)

		path, rawPath := construirDestino(rt.Destino, r)
		base := strings.TrimRight(destino.Path, "/")

		proxy := &httputil.ReverseProxy{
			Transport: c.transporte,
			Rewrite: func(pr *httputil.ProxyRequest) {
				pr.SetURL(destino) // esquema, host y query string
				pr.Out.URL.Path = base + path
				pr.Out.URL.RawPath = base + rawPath
				pr.SetXForwarded()
			},
			// El gateway es el único que pone CORS: se descartan las
			// cabeceras CORS que pudiera traer el servicio.
			ModifyResponse: func(resp *http.Response) error {
				for clave := range resp.Header {
					if strings.HasPrefix(http.CanonicalHeaderKey(clave), "Access-Control-") {
						resp.Header.Del(clave)
					}
				}
				return nil
			},
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				if errors.Is(err, context.Canceled) {
					return // el cliente cerró la conexión
				}
				var limite *http.MaxBytesError
				if errors.As(err, &limite) {
					responderError(w, http.StatusRequestEntityTooLarge, "cuerpo_muy_grande", "el cuerpo de la petición supera el tamaño máximo permitido")
					return
				}
				fmt.Printf("[Gateway] Error reenviando a %s (%s): %v\n", rt.Servicio, destino.Host, err)
				responderError(w, http.StatusBadGateway, "error_gateway", "el servicio "+rt.Servicio+" no respondió")
			},
		}
		proxy.ServeHTTP(w, r)
	}
}
