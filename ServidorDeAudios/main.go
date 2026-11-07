// Comando principal del ServidorDeAudios. Solo arranca y cablea las capas.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"audios.local/servidor-audios/consul"
	"audios.local/servidor-audios/controlador"
	"audios.local/servidor-audios/fachada"
	"audios.local/servidor-audios/repositorio"
)

const puertoHTTP = 8082

func main() {
	mux := construirMux()

	// Registro en Consul: solo si CONSUL_ADDR está definida. La función
	// devuelta lo desregistra al cerrar (Ctrl+C o SIGTERM, ver abajo).
	defer consul.Registrar("audios-service", puertoHTTP, "/health")()

	servidor := &http.Server{Addr: fmt.Sprintf(":%d", puertoHTTP), Handler: mux}
	ctx, detener := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer detener()

	errServidor := make(chan error, 1)
	go func() {
		fmt.Printf("ServidorDeAudios escuchando en %s (REST)...\n", servidor.Addr)
		errServidor <- servidor.ListenAndServe()
	}()

	select {
	case err := <-errServidor:
		panic(err)
	case <-ctx.Done():
		fmt.Println("Cerrando ServidorDeAudios...")
		ctxCierre, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelar()
		servidor.Shutdown(ctxCierre)
	}
}

func construirMux() *http.ServeMux {
	repo := repositorio.NuevoRepositorioAudios()
	fach := fachada.NuevaFachadaAudios(repo)
	ctrl := controlador.NuevoControladorAudios(fach)

	mux := http.NewServeMux()
	ctrl.RegistrarRutas(mux)
	return mux
}
