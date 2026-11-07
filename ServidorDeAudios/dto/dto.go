// Package dto agrupa los structs de respuesta JSON de este servidor.
package dto

// SubidaOkDTO se devuelve al terminar de guardar un mp3.
type SubidaOkDTO struct {
	Mensaje   string `json:"mensaje"`
	NombreMp3 string `json:"nombreMp3"`
}

// ErrorDTO es la respuesta estándar de error de los servicios REST: un
// código estable y un mensaje legible.
type ErrorDTO struct {
	Error   string `json:"error"`
	Mensaje string `json:"mensaje"`
}

// SaludDTO es la respuesta de GET /health (health check para Consul).
type SaludDTO struct {
	Status string `json:"status"`
}
