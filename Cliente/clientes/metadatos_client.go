// Package clientes agrupa los clientes REST que el cliente usa para hablar
// con el servidor de metadatos. Esta capa no conoce el
// menú ni la lógica de negocio: solo hace la llamada y devuelve DTOs o error.
package clientes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"cliente.local/cliente-audio/dto"
)

// MetadatosClient encapsula las llamadas REST al ServidorMetadataDeAudios.
type MetadatosClient struct {
	baseURL string
	http    *http.Client
}

// NuevoMetadatosClient recibe la URL base del servidor de metadatos,
// por ejemplo "http://localhost:8081".
func NuevoMetadatosClient(baseURL string) *MetadatosClient {
	return &MetadatosClient{baseURL: baseURL, http: &http.Client{}}
}

// leerJSON hace el GET y decodifica la respuesta en destino. El eco de la
// llamada lo imprime el servidor, no el cliente, para no ensuciar el menú. Si el status no es 200, decodifica el
// ErrorDTO y lo devuelve como error de Go.
func (c *MetadatosClient) leerJSON(ruta string, destino any) error {
	url := c.baseURL + ruta
	resp, err := c.http.Get(url)
	if err != nil {
		return fmt.Errorf("no se pudo contactar al servidor de metadatos: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errDTO dto.ErrorDTO
		if err := json.NewDecoder(resp.Body).Decode(&errDTO); err == nil && errDTO.Error != "" {
			return fmt.Errorf("%s", errDTO.Error)
		}
		return fmt.Errorf("el servidor respondió con estado %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(destino)
}

// ListarTipos consulta GET /tipos.
func (c *MetadatosClient) ListarTipos() ([]dto.TipoAudioDTO, error) {
	var tipos []dto.TipoAudioDTO
	if err := c.leerJSON("/tipos", &tipos); err != nil {
		return nil, err
	}
	return tipos, nil
}

// ListarAudiosPorTipo consulta GET /tipos/{id}/audios.
func (c *MetadatosClient) ListarAudiosPorTipo(tipoId int) ([]dto.AudioResumenDTO, error) {
	var audios []dto.AudioResumenDTO
	ruta := fmt.Sprintf("/tipos/%d/audios", tipoId)
	if err := c.leerJSON(ruta, &audios); err != nil {
		return nil, err
	}
	return audios, nil
}

// ObtenerDetalleAudio consulta GET /audios/{id}.
func (c *MetadatosClient) ObtenerDetalleAudio(audioId int) (dto.AudioDetalleDTO, error) {
	var detalle dto.AudioDetalleDTO
	ruta := fmt.Sprintf("/audios/%d", audioId)
	if err := c.leerJSON(ruta, &detalle); err != nil {
		return dto.AudioDetalleDTO{}, err
	}
	return detalle, nil
}
