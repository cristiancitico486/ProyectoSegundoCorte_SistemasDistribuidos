// Package dto contiene el mensaje que viaja por RabbitMQ hacia
// ServidorDeEstadisticas.
package dto

// ReproduccionDTO es el JSON que se publica cada vez que arranca una
// reproducción nueva.
type ReproduccionDTO struct {
	AudioId int    `json:"audioId"`
	Titulo  string `json:"titulo"`
	Tipo    string `json:"tipo"`
	Fecha   string `json:"fecha"`
	// UsuarioId identifica al usuario que reprodujo el audio (viene en la
	// petición gRPC). Con él el ServidorDeEstadisticas arma el historial.
	UsuarioId string `json:"usuarioId"`
}
