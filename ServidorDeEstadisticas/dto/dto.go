// Package dto contiene los mensajes de este servidor: el JSON que llega por
// la cola de RabbitMQ desde el ServidorDeStreaming y las respuestas REST.
package dto

// ReproduccionDTO es el JSON publicado cada vez que inicia una reproducción.
// Debe coincidir con el DTO que publica el ServidorDeStreaming.
type ReproduccionDTO struct {
	AudioId   int    `json:"audioId"`
	Titulo    string `json:"titulo"`
	Tipo      string `json:"tipo"`
	Fecha     string `json:"fecha"`
	UsuarioId string `json:"usuarioId"`
}

// ItemHistorialDTO es un audio del historial con la cantidad de veces que
// el usuario lo reprodujo.
type ItemHistorialDTO struct {
	AudioId            int    `json:"audioId"`
	Titulo             string `json:"titulo"`
	Tipo               string `json:"tipo"`
	Cantidad           int    `json:"cantidad"`
	UltimaReproduccion string `json:"ultimaReproduccion"`
}

// HistorialDTO es la respuesta de GET /estadisticas/historial/{usuarioId}.
type HistorialDTO struct {
	UsuarioId           string             `json:"usuarioId"`
	TotalReproducciones int                `json:"totalReproducciones"`
	Audios              []ItemHistorialDTO `json:"audios"`
}

// ErrorDTO es la respuesta estándar de error de los servicios REST.
type ErrorDTO struct {
	Error   string `json:"error"`
	Mensaje string `json:"mensaje"`
}

// SaludDTO es la respuesta de GET /health (health check para Consul).
type SaludDTO struct {
	Status string `json:"status"`
}
