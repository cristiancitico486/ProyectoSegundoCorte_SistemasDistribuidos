// Package reproductor define el contrato para reproducir un audio por
// streaming. La implementación que usa el cliente es StreamingGRPC
// (streaming.go), que habla con el ServidorDeStreaming.
package reproductor

// Reproductor reproduce un audio identificado por su id, título y tipo, y
// bloquea hasta que el audio termina o el usuario decide salir. Cuando
// retorna sin error, el controlador puede volver al menú principal.
// tipoNombre se reenvía al ServidorDeStreaming para el mensaje que se
// publica en la cola de estadísticas.
type Reproductor interface {
	Reproducir(audioId int, titulo string, tipoNombre string) error
}
