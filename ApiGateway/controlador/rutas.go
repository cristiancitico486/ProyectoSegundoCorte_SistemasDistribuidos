package controlador

import "gateway.local/api-gateway/descubrimiento"

const (
	// maxCuerpoJSON limita los POST con JSON (registrar metadatos, pagos).
	maxCuerpoJSON = 1 << 20 // 1 MB
	// maxSubidaMp3: 200 MB del mp3 más un margen para el multipart.
	maxSubidaMp3 = (200 << 20) + (1 << 20)
)

// ruta describe una ruta pública del gateway y a qué servicio se reenvía.
// Los {nombres} del Patron se sustituyen en Destino.
type ruta struct {
	Metodo    string
	Patron    string // ruta pública, ej. /api/tipos/{id}/audios
	Servicio  string // nombre en Consul
	Destino   string // ruta en el servicio, ej. /tipos/{id}/audios
	MaxCuerpo int64  // 0 = sin límite
}

// rutas es la tabla de contratos IA 1 v1.0, sección 1.
var rutas = []ruta{
	{"GET", "/api/tipos", descubrimiento.Metadata, "/tipos", 0},
	{"GET", "/api/tipos/{id}/audios", descubrimiento.Metadata, "/tipos/{id}/audios", 0},
	{"GET", "/api/tipos/{id}/campos", descubrimiento.Metadata, "/tipos/{id}/campos", 0},
	{"GET", "/api/audios/{id}", descubrimiento.Metadata, "/audios/{id}", 0},
	{"POST", "/api/audios", descubrimiento.Metadata, "/audios", maxCuerpoJSON},
	{"POST", "/api/audios/{id}/archivo", descubrimiento.Audios, "/audios/{id}", maxSubidaMp3},
	{"GET", "/api/historial/{usuarioId}", descubrimiento.Estadisticas, "/estadisticas/historial/{usuarioId}", 0},
	{"GET", "/api/planes", descubrimiento.Suscripciones, "/planes", 0},
	{"POST", "/api/pagos", descubrimiento.Suscripciones, "/pagos", maxCuerpoJSON},
	{"GET", "/api/suscripciones/{usuarioId}", descubrimiento.Suscripciones, "/suscripciones/{usuarioId}", 0},
	{"GET", "/ws/reacciones", descubrimiento.Reacciones, "/ws", 0},
}
