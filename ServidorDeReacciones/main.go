// Comando principal del ServidorDeReacciones (WebSocket). Solo arranca y
// cablea las capas: controlador -> fachada -> repositorio.
package main

import (
	"fmt"
	"net/http"

	"reacciones.local/servidor-reacciones/consul"
	"reacciones.local/servidor-reacciones/controlador"
	"reacciones.local/servidor-reacciones/fachada"
	"reacciones.local/servidor-reacciones/repositorio"
)

const puerto = ":8083"

func main() {
	// Registro en Consul: solo si CONSUL_ADDR está definida.
	defer consul.Registrar("reacciones-service", 8083, "/health")()

	repo := repositorio.NuevoRepositorioRegistro()
	fach := fachada.NuevaFachadaReacciones(repo)
	ctrl := controlador.NuevoControladorReacciones(fach)

	mux := http.NewServeMux()
	ctrl.RegistrarRutas(mux)

	fmt.Printf("ServidorDeReacciones escuchando en %s (WebSocket en /ws)...\n", puerto)
	if err := http.ListenAndServe(puerto, mux); err != nil {
		panic(err)
	}
}
