// Comando principal del ServidorDeStreaming. Solo arranca y cablea las
// capas: conexión a RabbitMQ, cliente REST hacia ServidorDeAudios, fachada,
// controlador gRPC y el listener.
package main

import (
	"fmt"
	"net"
	"os"

	"google.golang.org/grpc"

	"streaming.local/servidor-streaming/clientes"
	"streaming.local/servidor-streaming/consul"
	"streaming.local/servidor-streaming/controlador"
	"streaming.local/servidor-streaming/fachada"
	"streaming.local/servidor-streaming/pb"
	"streaming.local/servidor-streaming/publicador"
)

const (
	puertoGRPC          = ":50051"
	urlAudiosPorDefecto = "http://localhost:8082"
)

func main() {
	urlAudios := valorOEnv("AUDIOS_URL", urlAudiosPorDefecto)
	audiosClient := clientes.NuevoAudiosClient(urlAudios)

	pub, err := publicador.NuevoPublicador()
	if err != nil {
		// El streaming sigue funcionando aunque RabbitMQ esté caído: el
		// publicador reintenta la conexión en cada reproducción.
		fmt.Println("Advertencia: no se pudo conectar a RabbitMQ:", err)
		fmt.Println("El streaming seguirá funcionando; se reintentará al publicar.")
	}
	defer pub.Cerrar()

	// Registro en Consul: solo si CONSUL_ADDR está definida (check TCP).
	defer consul.Registrar("streaming-service", 50051, "")()

	fach := fachada.NuevaFachadaStreaming(audiosClient, pub)
	ctrl := controlador.NuevoControladorStreaming(fach)

	lis, err := net.Listen("tcp", puertoGRPC)
	if err != nil {
		panic(err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterAudioServiceServer(grpcServer, ctrl)

	fmt.Printf("ServidorDeStreaming escuchando en %s (gRPC)...\n", puertoGRPC)
	if err := grpcServer.Serve(lis); err != nil {
		panic(err)
	}
}

func valorOEnv(nombre, porDefecto string) string {
	if valor := os.Getenv(nombre); valor != "" {
		return valor
	}
	return porDefecto
}
