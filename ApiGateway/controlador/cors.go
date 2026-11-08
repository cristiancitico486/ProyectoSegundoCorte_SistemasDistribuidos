package controlador

import "net/http"

// ConCORS agrega las cabeceras CORS a todas las respuestas y contesta el
// preflight (OPTIONS) en cualquier ruta. El cliente web se abre como file://
// y su Origin es "null": "*" lo cubre porque no se usan credenciales.
func ConCORS(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cab := w.Header()
		cab.Set("Access-Control-Allow-Origin", "*")
		cab.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		cab.Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			// Se aceptan las cabeceras que el navegador anuncie y, siempre,
			// las habituales del cliente.
			permitidas := "Content-Type, Authorization, X-Requested-With, Idempotency-Key"
			if pedidas := r.Header.Get("Access-Control-Request-Headers"); pedidas != "" {
				permitidas += ", " + pedidas
			}
			cab.Set("Access-Control-Allow-Headers", permitidas)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		siguiente.ServeHTTP(w, r)
	})
}
