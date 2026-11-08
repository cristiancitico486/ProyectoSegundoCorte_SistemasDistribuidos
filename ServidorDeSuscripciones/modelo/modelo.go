// Package modelo contiene las entidades del dominio del servidor de
// suscripciones (la M de MVC).
package modelo

import "time"

// Plan es un plan de suscripción que el usuario puede pagar.
type Plan struct {
	Id           string
	Nombre       string
	Precio       int // en la moneda indicada, sin decimales (COP)
	Moneda       string
	DuracionDias int
	Descripcion  string
}

// Pago es un pago aprobado y almacenado. Cada línea de datos/pagos.jsonl es
// un Pago; el Token es la clave de idempotencia: un token se consume una
// sola vez.
type Pago struct {
	PagoId       string    `json:"pagoId"`
	Token        string    `json:"token"`
	UsuarioId    string    `json:"usuarioId"`
	PlanId       string    `json:"planId"`
	Plan         string    `json:"plan"`
	Monto        int       `json:"monto"`
	Moneda       string    `json:"moneda"`
	Fecha        time.Time `json:"fecha"`
	VigenteHasta time.Time `json:"vigenteHasta"`
}

var planes = []Plan{
	{Id: "basico", Nombre: "Básico", Precio: 9900, Moneda: "COP", DuracionDias: 30,
		Descripcion: "Catálogo completo en calidad estándar."},
	{Id: "estandar", Nombre: "Estándar", Precio: 19900, Moneda: "COP", DuracionDias: 30,
		Descripcion: "Catálogo completo en alta calidad."},
	{Id: "premium", Nombre: "Premium", Precio: 29900, Moneda: "COP", DuracionDias: 30,
		Descripcion: "Catálogo completo en la máxima calidad."},
}

// Planes devuelve los planes disponibles, en orden.
func Planes() []Plan {
	return append([]Plan(nil), planes...)
}

// BuscarPlan busca un plan por su id.
func BuscarPlan(id string) (Plan, bool) {
	for _, p := range planes {
		if p.Id == id {
			return p, true
		}
	}
	return Plan{}, false
}
