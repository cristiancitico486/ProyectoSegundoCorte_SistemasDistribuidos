// Comando principal del ServidorDeSuscripciones. Solo arranca y cablea las
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

	"suscripciones.local/servidor-suscripciones/consul"
	"suscripciones.local/servidor-suscripciones/controlador"
	"suscripciones.local/servidor-suscripciones/fachada"
	"suscripciones.local/servidor-suscripciones/repositorio"
)

const (
	rutaPagos  = "datos/pagos.jsonl"
	puertoHTTP = 8085
)

func main() {
	repo, err := repositorio.NuevoRepositorioPagos(rutaPagos)
	if err != nil {
		fmt.Println("No se pudo iniciar el repositorio de pagos:", err)
		os.Exit(1)
	}
	fach := fachada.NuevaFachadaSuscripciones(repo)
	ctrl := controlador.NuevoControladorSuscripciones(fach)
	mux := http.NewServeMux()
	ctrl.RegistrarRutas(mux)

	// Registro en Consul: solo si CONSUL_ADDR está definida. La función
	// devuelta lo desregistra al cerrar (Ctrl+C o SIGTERM, ver abajo).
	defer consul.Registrar("suscripciones-service", puertoHTTP, "/health")()

	servidor := &http.Server{
		Addr:              fmt.Sprintf(":%d", puertoHTTP),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	ctx, detener := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer detener()

	errServidor := make(chan error, 1)
	go func() {
		fmt.Printf("ServidorDeSuscripciones escuchando en %s (REST)...\n", servidor.Addr)
		errServidor <- servidor.ListenAndServe()
	}()

	select {
	case err := <-errServidor:
		panic(err)
	case <-ctx.Done():
		fmt.Println("Cerrando ServidorDeSuscripciones...")
		ctxCierre, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelar()
		servidor.Shutdown(ctxCierre)
		// Los pagos en curso van en goroutines propias: se espera a que
		// terminen para no perder un pago a medias.
		fach.EsperarPendientes(5 * time.Second)
	}
}
