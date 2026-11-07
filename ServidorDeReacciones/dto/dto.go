// Package dto define los mensajes JSON del canal WebSocket de reacciones
// (contrato v0.3, sección 4).
package dto

// Entrada es lo que envía el cliente. AudioId es puntero para distinguir
// "no vino" de 0.
type Entrada struct {
	Tipo     string `json:"tipo"`
	AudioId  *int   `json:"audioId"`
	Reaccion string `json:"reaccion"`
}

type Ack struct {
	Tipo      string `json:"tipo"` // "ack"
	Operacion string `json:"operacion"`
	AudioId   int    `json:"audioId"`
	Oyentes   int    `json:"oyentes"`
}

type ReaccionSalida struct {
	Tipo      string `json:"tipo"` // "reaccion"
	AudioId   int    `json:"audioId"`
	UsuarioId string `json:"usuarioId"`
	Reaccion  string `json:"reaccion"`
	Timestamp string `json:"timestamp"`
}

type Error struct {
	Tipo    string `json:"tipo"` // "error"
	Codigo  string `json:"codigo"`
	Mensaje string `json:"mensaje"`
}

// ErrorHTTP es el error REST común (usado solo antes del handshake).
type ErrorHTTP struct {
	Error   string `json:"error"`
	Mensaje string `json:"mensaje"`
}

type Salud struct {
	Status string `json:"status"`
}
