// Package dto agrupa las respuestas JSON que genera el propio gateway. Las
// respuestas de los servicios destino pasan sin modificarse.
package dto

// ErrorDTO es la respuesta estándar de error de los servicios REST.
type ErrorDTO struct {
	Error   string `json:"error"`
	Mensaje string `json:"mensaje"`
}

// SaludDTO es la respuesta de GET /health.
type SaludDTO struct {
	Status string `json:"status"`
}
