// Package clientes agrupa los clientes REST hacia los dos servidores que el
// administrador necesita: metadatos y audios. No conoce el menú ni decide
// nada de negocio, solo hace la llamada y devuelve DTOs o error.
package clientes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"administrador.local/administrador-cliente/dto"
)

type MetadatosClient struct {
	baseURL string
	http    *http.Client
}

func NuevoMetadatosClient(baseURL string) *MetadatosClient {
	return &MetadatosClient{baseURL: baseURL, http: &http.Client{Timeout: 10 * time.Second}}
}

// ListarTipos consulta GET /tipos, para que el administrador elija a qué
// tipo pertenece el audio nuevo.
func (c *MetadatosClient) ListarTipos() ([]dto.TipoAudioDTO, error) {
	url := c.baseURL + "/tipos"
	resp, err := c.http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("no se pudo contactar al servidor de metadatos: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errorDesdeRespuesta(resp)
	}
	var tipos []dto.TipoAudioDTO
	if err := json.NewDecoder(resp.Body).Decode(&tipos); err != nil {
		return nil, fmt.Errorf("respuesta inválida del servidor de metadatos: %w", err)
	}
	return tipos, nil
}

// RegistrarAudio consulta POST /audios: registra el título, tipo y
// metadatos del audio nuevo, y devuelve el id generado.
func (c *MetadatosClient) RegistrarAudio(entrada dto.AudioCrearDTO) (dto.AudioCreadoDTO, error) {
	url := c.baseURL + "/audios"
	cuerpo, err := json.Marshal(entrada)
	if err != nil {
		return dto.AudioCreadoDTO{}, fmt.Errorf("error preparando la solicitud: %w", err)
	}

	resp, err := c.http.Post(url, "application/json", bytes.NewReader(cuerpo))
	if err != nil {
		return dto.AudioCreadoDTO{}, fmt.Errorf("no se pudo contactar al servidor de metadatos: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return dto.AudioCreadoDTO{}, errorDesdeRespuesta(resp)
	}
	var creado dto.AudioCreadoDTO
	if err := json.NewDecoder(resp.Body).Decode(&creado); err != nil {
		return dto.AudioCreadoDTO{}, fmt.Errorf("respuesta inválida del servidor de metadatos: %w", err)
	}
	return creado, nil
}

// errorDesdeRespuesta decodifica un ErrorDTO del cuerpo de una respuesta no
// exitosa, o arma un mensaje genérico si el cuerpo no trae ese formato.
func errorDesdeRespuesta(resp *http.Response) error {
	var errDTO dto.ErrorDTO
	if err := json.NewDecoder(resp.Body).Decode(&errDTO); err == nil && errDTO.Error != "" {
		return fmt.Errorf("%s", errDTO.Error)
	}
	return fmt.Errorf("el servidor respondió con estado %d", resp.StatusCode)
}
