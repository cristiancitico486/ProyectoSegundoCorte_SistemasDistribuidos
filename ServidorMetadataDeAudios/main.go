// Comando principal del ServidorMetadataDeAudios. Solo arranca y cablea las
// capas: main no contiene lógica de negocio.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"metadatos.local/servidor-metadata/consul"
	"metadatos.local/servidor-metadata/controlador"
	"metadatos.local/servidor-metadata/fachada"
	"metadatos.local/servidor-metadata/repositorio"
)

const (
	rutaCatalogo = "datos/catalogo.json"
	puertoHTTP   = 8081
)

func main() {
	mux := construirMux()

	// Registro en Consul: solo si CONSUL_ADDR está definida. La función
	// devuelta lo desregistra al cerrar (Ctrl+C o SIGTERM, ver abajo).
	defer consul.Registrar("metadata-service", puertoHTTP, "/health")()

	servidor := &http.Server{Addr: fmt.Sprintf(":%d", puertoHTTP), Handler: mux}
	ctx, detener := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer detener()

	errServidor := make(chan error, 1)
	go func() {
		fmt.Printf("ServidorMetadataDeAudios escuchando en %s (REST)...\n", servidor.Addr)
		errServidor <- servidor.ListenAndServe()
	}()

	select {
	case err := <-errServidor:
		panic(err)
	case <-ctx.Done():
		fmt.Println("Cerrando ServidorMetadataDeAudios...")
		ctxCierre, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelar()
		servidor.Shutdown(ctxCierre)
	}
}

// construirMux cablea repositorio -> fachada -> controlador y registra las rutas.
func construirMux() *http.ServeMux {
	repo := repositorio.ObtenerRepositorio(rutaCatalogo)
	fach := fachada.NuevaFachadaMetadatos(repo)
	ctrl := controlador.NuevoControladorMetadatos(fach)

	mux := http.NewServeMux()
	ctrl.RegistrarRutas(mux)
	return mux
}
