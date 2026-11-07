// Package clientes agrupa las llamadas de red que hace este servidor hacia
// otros servicios: REST hacia ServidorDeAudios para leer el mp3.
package clientes

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

type AudiosClient struct {
	baseURL string
	http    *http.Client
}

func NuevoAudiosClient(baseURL string) *AudiosClient {
	return &AudiosClient{baseURL: baseURL, http: &http.Client{Timeout: 30 * time.Second}}
}

// AbrirArchivo llama a GET /audios/{id}/archivo en el ServidorDeAudios y
// devuelve el cuerpo de la respuesta como io.ReadCloser, listo para leerlo
// en chunks. El llamador es responsable de cerrarlo.
//
// A propósito NO se lee todo el body aquí: el requerimiento pide no cargar
// el mp3 completo en memoria antes de reenviarlo, así que este cliente solo
// abre la conexión y deja que la fachada la lea de a 32KB mientras la
// reenvía por gRPC.
func (c *AudiosClient) AbrirArchivo(audioId int) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/audios/%d/archivo", c.baseURL, audioId)
	fmt.Printf("[REST] ServidorDeStreaming llama GET %s\n", url)

	resp, err := c.http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("no se pudo contactar al servidor de audios: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("el servidor de audios respondió con estado %d para audioId=%d", resp.StatusCode, audioId)
	}
	return resp.Body, nil
}
