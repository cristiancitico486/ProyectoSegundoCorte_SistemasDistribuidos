// Comando principal del Cliente. Solo arranca y cablea las capas: main no
// contiene lógica de negocio ni de presentación.
package main

import (
	"os"

	"cliente.local/cliente-audio/clientes"
	"cliente.local/cliente-audio/controlador"
	"cliente.local/cliente-audio/fachada"
	"cliente.local/cliente-audio/reproductor"
)

// Valores por defecto en localhost; sobreescribibles con variables de
// entorno si los servidores corren en otra máquina de la red.
const (
	urlMetadatosPorDefecto = "http://localhost:8081"
	urlStreamingPorDefecto = "localhost:50051"
)

func main() {
	urlMetadatos := valorOEnv("METADATOS_URL", urlMetadatosPorDefecto)
	urlStreaming := valorOEnv("STREAMING_URL", urlStreamingPorDefecto)

	metadatosClient := clientes.NuevoMetadatosClient(urlMetadatos)
	rep := reproductor.NuevoStreamingGRPC(urlStreaming)
	f := fachada.NuevaFachadaCliente(metadatosClient, rep)

	controlador.Iniciar(f)
}

func valorOEnv(nombre, porDefecto string) string {
	if valor := os.Getenv(nombre); valor != "" {
		return valor
	}
	return porDefecto
}
