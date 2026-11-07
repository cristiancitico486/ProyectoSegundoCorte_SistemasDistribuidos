// Package controlador expone los handlers REST (la C de MVC). Cada handler
// imprime un eco por printf, como exige el requerimiento, y delega toda la
// lógica en la fachada.
package controlador

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"metadatos.local/servidor-metadata/dto"
	"metadatos.local/servidor-metadata/fachada"
)

// maxCuerpoJSON limita el tamaño del body de POST /audios.
const maxCuerpoJSON = 1 << 20 // 1 MB

type ControladorMetadatos struct {
	fachada *fachada.FachadaMetadatos
}

func NuevoControladorMetadatos(f *fachada.FachadaMetadatos) *ControladorMetadatos {
	return &ControladorMetadatos{fachada: f}
}

// RegistrarRutas cablea las rutas REST en el mux recibido.
func (c *ControladorMetadatos) RegistrarRutas(mux *http.ServeMux) {
	mux.HandleFunc("GET /tipos", c.ListarTipos)
	mux.HandleFunc("GET /tipos/{id}/audios", c.ListarAudiosPorTipo)
	mux.HandleFunc("GET /tipos/{id}/campos", c.ListarCamposPorTipo)
	mux.HandleFunc("GET /audios/{id}", c.ObtenerDetalleAudio)
	mux.HandleFunc("POST /audios", c.RegistrarAudio)
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

// responderErrorFachada traduce un error de la fachada a su respuesta HTTP:
// los errores de negocio conocidos llevan su código; cualquier otro es un
// error interno (por ejemplo, falló el guardado en disco).
func responderErrorFachada(w http.ResponseWriter, err error) {
	var negocio *fachada.ErrorNegocio
	if !errors.As(err, &negocio) {
		fmt.Printf("[REST] Error interno: %v\n", err)
		responderError(w, http.StatusInternalServerError, "error_interno", "no se pudo completar la operación")
		return
	}
	status := http.StatusBadRequest
	switch negocio.Codigo {
	case fachada.ErrTipoNoEncontrado.Codigo, fachada.ErrAudioNoEncontrado.Codigo:
		status = http.StatusNotFound
	}
	responderJSON(w, status, dto.ErrorDTO{Error: negocio.Codigo, Mensaje: negocio.Mensaje, Faltantes: negocio.Faltantes})
}

// ListarTipos atiende GET /tipos.
func (c *ControladorMetadatos) ListarTipos(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[REST] GET /tipos llamado desde %s\n", r.RemoteAddr)
	responderJSON(w, http.StatusOK, c.fachada.ListarTipos())
}

// ListarAudiosPorTipo atiende GET /tipos/{id}/audios.
func (c *ControladorMetadatos) ListarAudiosPorTipo(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	fmt.Printf("[REST] GET /tipos/%s/audios llamado desde %s\n", idStr, r.RemoteAddr)

	tipoId, err := strconv.Atoi(idStr)
	if err != nil {
		responderError(w, http.StatusBadRequest, "id_invalido", "id de tipo inválido")
		return
	}

	audios, err := c.fachada.ListarAudiosPorTipo(tipoId)
	if err != nil {
		responderErrorFachada(w, err)
		return
	}
	responderJSON(w, http.StatusOK, audios)
}

// ListarCamposPorTipo atiende GET /tipos/{id}/campos: los metadatos que
// pide el tipo (obligatorios primero), para armar el formulario del
// administrador.
func (c *ControladorMetadatos) ListarCamposPorTipo(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	fmt.Printf("[REST] GET /tipos/%s/campos llamado desde %s\n", idStr, r.RemoteAddr)

	tipoId, err := strconv.Atoi(idStr)
	if err != nil {
		responderError(w, http.StatusBadRequest, "id_invalido", "id de tipo inválido")
		return
	}

	campos, err := c.fachada.ConsultarCampos(tipoId)
	if err != nil {
		responderErrorFachada(w, err)
		return
	}
	responderJSON(w, http.StatusOK, campos)
}

// ObtenerDetalleAudio atiende GET /audios/{id}.
func (c *ControladorMetadatos) ObtenerDetalleAudio(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	fmt.Printf("[REST] GET /audios/%s llamado desde %s\n", idStr, r.RemoteAddr)

	id, err := strconv.Atoi(idStr)
	if err != nil {
		responderError(w, http.StatusBadRequest, "id_invalido", "id de audio inválido")
		return
	}

	detalle, err := c.fachada.ObtenerDetalleAudio(id)
	if err != nil {
		responderErrorFachada(w, err)
		return
	}
	responderJSON(w, http.StatusOK, detalle)
}

// RegistrarAudio atiende POST /audios: valida y registra los metadatos y
// devuelve el id que el cliente usará para subir el mp3.
func (c *ControladorMetadatos) RegistrarAudio(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[REST] POST /audios llamado desde %s\n", r.RemoteAddr)

	var entrada dto.AudioCrearDTO
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxCuerpoJSON)).Decode(&entrada); err != nil {
		responderError(w, http.StatusBadRequest, "cuerpo_invalido", "cuerpo JSON inválido")
		return
	}

	creado, err := c.fachada.RegistrarAudio(entrada)
	if err != nil {
		responderErrorFachada(w, err)
		return
	}
	fmt.Printf("[REST] Audio registrado con id=%d\n", creado.Id)
	responderJSON(w, http.StatusCreated, creado)
}

// Salud atiende GET /health, que consulta Consul periódicamente. No imprime
// eco para no llenar la consola.
func (c *ControladorMetadatos) Salud(w http.ResponseWriter, r *http.Request) {
	responderJSON(w, http.StatusOK, dto.SaludDTO{Status: "ok"})
}
