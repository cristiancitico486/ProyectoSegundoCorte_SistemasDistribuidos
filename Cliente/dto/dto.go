// Package dto contiene los structs que el cliente recibe serializados como
// JSON desde el ServidorMetadataDeAudios. Deben coincidir con los DTO que
// expone ese servidor.
package dto

// TipoAudioDTO llega en la respuesta de GET /tipos.
type TipoAudioDTO struct {
	Id     int    `json:"id"`
	Nombre string `json:"nombre"`
}

// AudioResumenDTO llega en la respuesta de GET /tipos/{id}/audios.
type AudioResumenDTO struct {
	Id     int    `json:"id"`
	Titulo string `json:"titulo"`
}

// AudioDetalleDTO llega en la respuesta de GET /audios/{id}.
type AudioDetalleDTO struct {
	Id        int               `json:"id"`
	TipoId    int               `json:"tipoId"`
	Titulo    string            `json:"titulo"`
	Metadatos map[string]string `json:"metadatos"`
	NombreMp3 string            `json:"nombreMp3"`
}

// ErrorDTO es la respuesta estándar de error del servidor de metadatos.
type ErrorDTO struct {
	Error string `json:"error"`
}
