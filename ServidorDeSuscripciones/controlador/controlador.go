// Package controlador expone los handlers REST (la C de MVC). La lógica de
// negocio, incluida la idempotencia de los pagos, está en la fachada.
package controlador

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"suscripciones.local/servidor-suscripciones/dto"
	"suscripciones.local/servidor-suscripciones/fachada"
)

// maxCuerpoJSON limita el tamaño del body de POST /pagos.
const maxCuerpoJSON = 1 << 20 // 1 MB

type ControladorSuscripciones struct {
	fachada *fachada.FachadaSuscripciones
}

func NuevoControladorSuscripciones(f *fachada.FachadaSuscripciones) *ControladorSuscripciones {
	return &ControladorSuscripciones{fachada: f}
}

// RegistrarRutas cablea las rutas REST en el mux recibido.
func (c *ControladorSuscripciones) RegistrarRutas(mux *http.ServeMux) {
	mux.HandleFunc("GET /planes", c.ListarPlanes)
	mux.HandleFunc("POST /pagos", c.RegistrarPago)
	mux.HandleFunc("GET /suscripciones/{usuarioId}", c.ConsultarSuscripcion)
	mux.HandleFunc("GET /health", c.Salud)
}

func responderJSON(w http.ResponseWriter, status int, cuerpo any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(cuerpo)
}

func responderError(w http.ResponseWriter, status int, codigo, mensaje string) {
	responderJSON(w, status, dto.ErrorDTO{Error: codigo, Mensaje: mensaje})
}

// responderErrorFachada traduce un error de la fachada a su respuesta HTTP.
func responderErrorFachada(w http.ResponseWriter, err error) {
	var negocio *fachada.ErrorNegocio
	if !errors.As(err, &negocio) {
		fmt.Printf("[REST] Error interno: %v\n", err)
		responderError(w, http.StatusInternalServerError, "error_interno", "no se pudo completar la operación")
		return
	}
	status := http.StatusBadRequest
	switch negocio.Codigo {
	case "pago_rechazado":
		status = http.StatusPaymentRequired
	case "token_en_conflicto":
		status = http.StatusConflict
	case "error_interno":
		status = http.StatusInternalServerError
	}
	responderError(w, status, negocio.Codigo, negocio.Mensaje)
}

// ListarPlanes atiende GET /planes.
func (c *ControladorSuscripciones) ListarPlanes(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[REST] GET /planes llamado desde %s\n", r.RemoteAddr)
	responderJSON(w, http.StatusOK, c.fachada.ListarPlanes())
}

// RegistrarPago atiende POST /pagos: 201 si el pago es nuevo, 200 si el
// token ya se había usado (resultado original, sin cobrar de nuevo).
func (c *ControladorSuscripciones) RegistrarPago(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[REST] POST /pagos llamado desde %s\n", r.RemoteAddr)

	var entrada dto.PagoSolicitudDTO
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxCuerpoJSON)).Decode(&entrada); err != nil {
		responderError(w, http.StatusBadRequest, "cuerpo_invalido", "cuerpo JSON inválido")
		return
	}

	resultado, err := c.fachada.ProcesarPago(r.Context(), entrada)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		// El cliente cerró la conexión (por ejemplo, venció su timeout). El
		// pago NO se cancela: sigue procesándose y quedará almacenado.
		fmt.Println("[Pago] El cliente cerró la conexión antes de recibir la respuesta: el pago sigue procesándose")
		return
	}
	if err != nil {
		responderErrorFachada(w, err)
		return
	}
	if resultado.Repetido {
		responderJSON(w, http.StatusOK, resultado)
		return
	}
	responderJSON(w, http.StatusCreated, resultado)
}

// ConsultarSuscripcion atiende GET /suscripciones/{usuarioId}.
func (c *ControladorSuscripciones) ConsultarSuscripcion(w http.ResponseWriter, r *http.Request) {
	usuarioId := r.PathValue("usuarioId")
	fmt.Printf("[REST] GET /suscripciones/%s llamado desde %s\n", usuarioId, r.RemoteAddr)

	respuesta, err := c.fachada.ConsultarSuscripcion(usuarioId)
	if err != nil {
		responderErrorFachada(w, err)
		return
	}
	responderJSON(w, http.StatusOK, respuesta)
}

// Salud atiende GET /health, que consulta Consul periódicamente. No imprime
// eco para no llenar la consola.
func (c *ControladorSuscripciones) Salud(w http.ResponseWriter, r *http.Request) {
	responderJSON(w, http.StatusOK, dto.SaludDTO{Status: "ok"})
}
