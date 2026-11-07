package controlador

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"estadisticas.local/servidor-estadisticas/dto"
	"estadisticas.local/servidor-estadisticas/fachada"
)

// RegistrarRutas cablea las rutas REST de consulta en el mux recibido. El
// API Gateway las consume; el consumo de la cola sigue siendo asíncrono.
func (c *ControladorEstadisticas) RegistrarRutas(mux *http.ServeMux) {
	mux.HandleFunc("GET /estadisticas/historial/{usuarioId}", c.ObtenerHistorial)
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

// ObtenerHistorial atiende GET /estadisticas/historial/{usuarioId}.
func (c *ControladorEstadisticas) ObtenerHistorial(w http.ResponseWriter, r *http.Request) {
	usuarioId := r.PathValue("usuarioId")
	fmt.Printf("[REST] GET /estadisticas/historial/%s llamado desde %s\n", usuarioId, r.RemoteAddr)

	historial, err := c.fachada.ConsultarHistorial(usuarioId)
	if errors.Is(err, fachada.ErrUsuarioInvalido) {
		responderError(w, http.StatusBadRequest, "usuario_invalido", err.Error())
		return
	}
	if err != nil {
		responderError(w, http.StatusInternalServerError, "error_interno", err.Error())
		return
	}
	responderJSON(w, http.StatusOK, historial)
}

// Salud atiende GET /health, que consulta Consul periódicamente. No imprime
// eco para no llenar la consola.
func (c *ControladorEstadisticas) Salud(w http.ResponseWriter, r *http.Request) {
	responderJSON(w, http.StatusOK, dto.SaludDTO{Status: "ok"})
}
