// Comando principal del API Gateway. Cablea descubrimiento -> controlador y
// arranca el servidor HTTP en :9000.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gateway.local/api-gateway/controlador"
	"gateway.local/api-gateway/descubrimiento"
)

const puertoHTTP = 9000

// urlsPorDefecto se usan solo cuando no hay CONSUL_ADDR.
var urlsPorDefecto = []struct{ servicio, variable, valor string }{
	{descubrimiento.Metadata, "METADATA_URL", "http://localhost:8081"},
	{descubrimiento.Audios, "AUDIOS_URL", "http://localhost:8082"},
	{descubrimiento.Reacciones, "REACCIONES_URL", "http://localhost:8083"},
	{descubrimiento.Estadisticas, "ESTADISTICAS_URL", "http://localhost:8084"},
	{descubrimiento.Suscripciones, "SUSCRIPCIONES_URL", "http://localhost:8085"},
}

func construirResolutor() (descubrimiento.Resolutor, error) {
	if direccion := os.Getenv("CONSUL_ADDR"); direccion != "" {
		fmt.Printf("[Gateway] Descubrimiento por Consul en %s\n", direccion)
		return descubrimiento.NuevoConsul(direccion), nil
	}
	urls := map[string]string{}
	for _, d := range urlsPorDefecto {
		valor := os.Getenv(d.variable)
		if valor == "" {
			valor = d.valor
		}
		urls[d.servicio] = valor
		fmt.Printf("[Gateway] %s -> %s\n", d.servicio, valor)
	}
	fmt.Println("[Gateway] CONSUL_ADDR no definida: se usan las URLs fijas de arriba")
	return descubrimiento.NuevoEstatico(urls)
}

func main() {
	resolutor, err := construirResolutor()
	if err != nil {
		fmt.Println("Configuración inválida:", err)
		os.Exit(1)
	}
	ctrl := controlador.NuevoControladorGateway(resolutor)

	// Solo ReadHeaderTimeout: un ReadTimeout o WriteTimeout cortaría las
	// conexiones WebSocket (duran minutos) y la subida de mp3 de hasta 200 MB.
	servidor := &http.Server{
		Addr:              fmt.Sprintf(":%d", puertoHTTP),
		Handler:           ctrl.Manejador(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	ctx, detener := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer detener()

	errServidor := make(chan error, 1)
	go func() {
		fmt.Printf("API Gateway escuchando en %s ...\n", servidor.Addr)
		errServidor <- servidor.ListenAndServe()
	}()

	select {
	case err := <-errServidor:
		panic(err)
	case <-ctx.Done():
		fmt.Println("Cerrando API Gateway...")
		ctxCierre, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelar()
		servidor.Shutdown(ctxCierre)
	}
}
