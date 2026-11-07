// Comando principal del ServidorDeEstadisticas. Solo arranca y cablea las
// capas: consumidor de la cola -> controlador -> fachada -> repositorio.
package main

import (
	"fmt"
	"net/http"

	"estadisticas.local/servidor-estadisticas/consul"
	"estadisticas.local/servidor-estadisticas/consumidor"
	"estadisticas.local/servidor-estadisticas/controlador"
	"estadisticas.local/servidor-estadisticas/fachada"
	"estadisticas.local/servidor-estadisticas/repositorio"
)

const (
	rutaReproducciones = "datos/reproducciones.jsonl"
	puertoREST         = ":8084"
)

func main() {
	repo, err := repositorio.NuevoRepositorioReproducciones(rutaReproducciones)
	if err != nil {
		panic(err)
	}
	// Registro en Consul: solo si CONSUL_ADDR está definida.
	defer consul.Registrar("estadisticas-service", 8084, "/health")()

	fach := fachada.NuevaFachadaEstadisticas(repo)
	ctrl := controlador.NuevoControladorEstadisticas(fach, consumidor.NuevoConsumidorCola())

	// REST de consulta (historial y resumen) en paralelo al consumo de la cola.
	mux := http.NewServeMux()
	ctrl.RegistrarRutas(mux)
	go func() {
		fmt.Printf("ServidorDeEstadisticas escuchando en %s (REST)...\n", puertoREST)
		if err := http.ListenAndServe(puertoREST, mux); err != nil {
			panic(err)
		}
	}()

	fmt.Println("ServidorDeEstadisticas iniciado (consume la cola de reproducciones)...")
	ctrl.Iniciar()
}
