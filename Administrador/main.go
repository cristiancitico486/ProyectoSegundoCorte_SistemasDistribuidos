// Comando principal del Administrador. Solo arranca y cablea las capas.
// Es un binario aparte del Cliente: el administrador ya no es un modo de
// ./cliente, es su propio programa.
package main

import (
	"os"

	"administrador.local/administrador-cliente/clientes"
	"administrador.local/administrador-cliente/controlador"
	"administrador.local/administrador-cliente/fachada"
)

// Valores por defecto en localhost; sobreescribibles con variables de
// entorno si los servidores corren en otra máquina de la red.
const (
	urlMetadatosPorDefecto = "http://localhost:8081"
	urlAudiosPorDefecto    = "http://localhost:8082"
)

func main() {
	urlMetadatos := valorOEnv("METADATOS_URL", urlMetadatosPorDefecto)
	urlAudios := valorOEnv("AUDIOS_URL", urlAudiosPorDefecto)

	metadatosClient := clientes.NuevoMetadatosClient(urlMetadatos)
	audiosClient := clientes.NuevoAudiosClient(urlAudios)
	f := fachada.NuevaFachadaAdministrador(metadatosClient, audiosClient)

	controlador.Iniciar(f)
}

func valorOEnv(nombre, porDefecto string) string {
	if valor := os.Getenv(nombre); valor != "" {
		return valor
	}
	return porDefecto
}
