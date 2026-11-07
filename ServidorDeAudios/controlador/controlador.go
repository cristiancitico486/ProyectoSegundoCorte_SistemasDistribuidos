// Package controlador expone los handlers REST de este servidor. Cada
// handler imprime un eco por printf, como exige el requerimiento.
package controlador

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"audios.local/servidor-audios/dto"
	"audios.local/servidor-audios/fachada"
)

// tamanioMaximoSubida limita el tamaño de la subida a 200MB, holgado para
// audios completos de música, podcasts o audiolibros. Se aplica sobre el
// body completo (el multipart agrega un pequeño margen de encabezados).
const tamanioMaximoSubida = 200 << 20

type ControladorAudios struct {
	fachada *fachada.FachadaAudios
}

func NuevoControladorAudios(f *fachada.FachadaAudios) *ControladorAudios {
	return &ControladorAudios{fachada: f}
}

func (c *ControladorAudios) RegistrarRutas(mux *http.ServeMux) {
	mux.HandleFunc("POST /audios/{id}", c.SubirAudio)
	mux.HandleFunc("GET /audios/{id}/archivo", c.ObtenerArchivo)
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

// idDeRuta lee y valida el {id} de la ruta (entero positivo).
func idDeRuta(r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	return id, err == nil && id > 0
}

// SubirAudio atiende POST /audios/{id}: recibe el
// mp3 como multipart/form-data en el campo "archivo" y lo guarda en disco.
// El id ya viene asignado por el ServidorMetadataDeAudios.
func (c *ControladorAudios) SubirAudio(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	fmt.Printf("[REST] POST /audios/%s llamado desde %s\n", idStr, r.RemoteAddr)

	id, ok := idDeRuta(r)
	if !ok {
		responderError(w, http.StatusBadRequest, "id_invalido", "el id debe ser un entero positivo")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, tamanioMaximoSubida+(1<<20))
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		var limite *http.MaxBytesError
		if errors.As(err, &limite) {
			responderError(w, http.StatusRequestEntityTooLarge, "archivo_muy_grande", "el archivo supera el máximo de 200 MB")
			return
		}
		responderError(w, http.StatusBadRequest, "formulario_invalido", "no se pudo leer el formulario multipart: "+err.Error())
		return
	}
	archivo, _, err := r.FormFile("archivo")
	if err != nil {
		responderError(w, http.StatusBadRequest, "archivo_requerido", "falta el campo 'archivo' con el mp3")
		return
	}
	defer archivo.Close()

	data, err := io.ReadAll(archivo)
	if err != nil {
		responderError(w, http.StatusInternalServerError, "error_interno", "error leyendo el archivo recibido")
		return
	}

	nombreMp3, err := c.fachada.GuardarAudio(id, data)
	switch {
	case errors.Is(err, fachada.ErrArchivoVacio):
		responderError(w, http.StatusBadRequest, "archivo_vacio", err.Error())
		return
	case errors.Is(err, fachada.ErrNoEsMp3):
		responderError(w, http.StatusBadRequest, "archivo_no_mp3", err.Error())
		return
	case err != nil:
		fmt.Printf("[REST] Error interno guardando el audio %d: %v\n", id, err)
		responderError(w, http.StatusInternalServerError, "error_interno", "no se pudo guardar el archivo")
		return
	}

	fmt.Printf("[REST] Audio %s guardado en disco (%d bytes)\n", nombreMp3, len(data))
	responderJSON(w, http.StatusCreated, dto.SubidaOkDTO{
		Mensaje:   "audio guardado correctamente",
		NombreMp3: nombreMp3,
	})
}

// ObtenerArchivo atiende GET /audios/{id}/archivo. La consume el
// ServidorDeStreaming para leer el mp3 y reenviarlo en chunks por gRPC.
func (c *ControladorAudios) ObtenerArchivo(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	fmt.Printf("[REST] GET /audios/%s/archivo llamado desde %s\n", idStr, r.RemoteAddr)

	id, ok := idDeRuta(r)
	if !ok {
		responderError(w, http.StatusBadRequest, "id_invalido", "el id debe ser un entero positivo")
		return
	}

	ruta, err := c.fachada.ObtenerRutaAudio(id)
	if err != nil {
		if errors.Is(err, fachada.ErrNoEncontrado) {
			responderError(w, http.StatusNotFound, "audio_no_encontrado", err.Error())
			return
		}
		responderError(w, http.StatusInternalServerError, "error_interno", "no se pudo leer el archivo")
		return
	}

	w.Header().Set("Content-Type", "audio/mpeg")
	http.ServeFile(w, r, ruta)
}

// Salud atiende GET /health, que consulta Consul periódicamente. No imprime
// eco para no llenar la consola.
func (c *ControladorAudios) Salud(w http.ResponseWriter, r *http.Request) {
	responderJSON(w, http.StatusOK, dto.SaludDTO{Status: "ok"})
}
