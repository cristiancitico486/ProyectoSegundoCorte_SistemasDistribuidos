package clientes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"administrador.local/administrador-cliente/dto"
)

type AudiosClient struct {
	baseURL string
	http    *http.Client
}

func NuevoAudiosClient(baseURL string) *AudiosClient {
	// Timeout amplio: subir un mp3 completo puede tardar más que una
	// consulta REST normal, sobre todo con archivos grandes.
	return &AudiosClient{baseURL: baseURL, http: &http.Client{Timeout: 2 * time.Minute}}
}

// SubirMp3 lee el archivo local en rutaLocal y lo envía como
// multipart/form-data (campo "archivo") a POST /audios/{id}.
func (c *AudiosClient) SubirMp3(id int, rutaLocal string) (dto.SubidaOkDTO, error) {
	archivo, err := os.Open(rutaLocal)
	if err != nil {
		return dto.SubidaOkDTO{}, fmt.Errorf("no se pudo abrir el archivo local: %w", err)
	}
	defer archivo.Close()

	var cuerpo bytes.Buffer
	writer := multipart.NewWriter(&cuerpo)
	parte, err := writer.CreateFormFile("archivo", filepath.Base(rutaLocal))
	if err != nil {
		return dto.SubidaOkDTO{}, fmt.Errorf("error preparando la subida: %w", err)
	}
	if _, err := io.Copy(parte, archivo); err != nil {
		return dto.SubidaOkDTO{}, fmt.Errorf("error leyendo el archivo local: %w", err)
	}
	if err := writer.Close(); err != nil {
		return dto.SubidaOkDTO{}, fmt.Errorf("error preparando la subida: %w", err)
	}

	url := fmt.Sprintf("%s/audios/%d", c.baseURL, id)
	req, err := http.NewRequest(http.MethodPost, url, &cuerpo)
	if err != nil {
		return dto.SubidaOkDTO{}, fmt.Errorf("error preparando la solicitud: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return dto.SubidaOkDTO{}, fmt.Errorf("no se pudo contactar al servidor de audios: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return dto.SubidaOkDTO{}, errorDesdeRespuesta(resp)
	}
	var ok dto.SubidaOkDTO
	if err := json.NewDecoder(resp.Body).Decode(&ok); err != nil {
		return dto.SubidaOkDTO{}, fmt.Errorf("respuesta inválida del servidor de audios: %w", err)
	}
	return ok, nil
}
