// Package dto agrupa los structs que viajan serializados como JSON por REST.
package dto

// PlanDTO es un plan de GET /planes.
type PlanDTO struct {
	Id           string `json:"id"`
	Nombre       string `json:"nombre"`
	Precio       int    `json:"precio"`
	Moneda       string `json:"moneda"`
	DuracionDias int    `json:"duracionDias"`
	Descripcion  string `json:"descripcion"`
}

// PagoSolicitudDTO es el body de POST /pagos. Simulacion es opcional y solo
// sirve para la demostración: "ninguna" (o vacío), "demora" o "rechazo".
type PagoSolicitudDTO struct {
	Token      string `json:"token"`
	UsuarioId  string `json:"usuarioId"`
	PlanId     string `json:"planId"`
	Simulacion string `json:"simulacion"`
}

// PagoRespuestaDTO es la respuesta de POST /pagos. Repetido indica que el
// token ya se había usado y se devuelve el resultado original sin cobrar de
// nuevo.
type PagoRespuestaDTO struct {
	Token        string `json:"token"`
	PagoId       string `json:"pagoId"`
	UsuarioId    string `json:"usuarioId"`
	PlanId       string `json:"planId"`
	Plan         string `json:"plan"`
	Monto        int    `json:"monto"`
	Moneda       string `json:"moneda"`
	Estado       string `json:"estado"`
	Almacenado   bool   `json:"almacenado"`
	Repetido     bool   `json:"repetido"`
	Fecha        string `json:"fecha"`
	VigenteHasta string `json:"vigenteHasta"`
}

// PagoHistorialDTO es un pago dentro de GET /suscripciones/{usuarioId}.
type PagoHistorialDTO struct {
	PagoId string `json:"pagoId"`
	Token  string `json:"token"`
	PlanId string `json:"planId"`
	Monto  int    `json:"monto"`
	Moneda string `json:"moneda"`
	Fecha  string `json:"fecha"`
}

// SuscripcionDTO es la respuesta de GET /suscripciones/{usuarioId}.
type SuscripcionDTO struct {
	UsuarioId    string             `json:"usuarioId"`
	Activa       bool               `json:"activa"`
	PlanId       string             `json:"planId"`
	Plan         string             `json:"plan"`
	VigenteHasta string             `json:"vigenteHasta,omitempty"`
	Pagos        []PagoHistorialDTO `json:"pagos"`
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
