// Package fachada contiene la lógica de negocio del streaming: pedir el
// archivo al ServidorDeAudios, reenviarlo en chunks por gRPC, y publicar en
// la cola de estadísticas al iniciar la reproducción.
package fachada

import (
	"fmt"
	"io"
	"time"

	"streaming.local/servidor-streaming/clientes"
	"streaming.local/servidor-streaming/dto"
	"streaming.local/servidor-streaming/pb"
	"streaming.local/servidor-streaming/publicador"
)

const tamanioChunk = 32 * 1024 // 32 KB por fragmento

type FachadaStreaming struct {
	audios     *clientes.AudiosClient
	publicador *publicador.PublicadorReproducciones
}

func NuevaFachadaStreaming(audios *clientes.AudiosClient, pub *publicador.PublicadorReproducciones) *FachadaStreaming {
	return &FachadaStreaming{audios: audios, publicador: pub}
}

// ReproducirAudio abre el mp3 en el ServidorDeAudios, publica el evento de
// reproducción en la cola y reenvía el archivo en chunks de 32 KB por el
// stream gRPC, sin cargarlo completo en memoria. El evento solo se publica
// si el archivo existe, para no contar reproducciones que fallaron.
func (f *FachadaStreaming) ReproducirAudio(req *pb.AudioRequest, stream pb.AudioService_AudioStreamServer) error {
	audioId := int(req.AudioId)
	fmt.Printf("[gRPC] AudioStream llamado audioId=%d titulo=%q usuarioId=%q\n", audioId, req.Titulo, req.UsuarioId)

	archivo, err := f.audios.AbrirArchivo(audioId)
	if err != nil {
		return fmt.Errorf("no se pudo obtener el archivo: %w", err)
	}
	defer archivo.Close()

	f.publicarReproduccion(audioId, req)

	buf := make([]byte, tamanioChunk)
	totalChunks := 0
	for {
		n, err := archivo.Read(buf)
		if n > 0 {
			totalChunks++
			if errEnvio := stream.Send(&pb.AudioChunk{Data: buf[:n]}); errEnvio != nil {
				fmt.Printf("[gRPC] Streaming de audioId=%d interrumpido tras %d chunks\n", audioId, totalChunks)
				return fmt.Errorf("error enviando el fragmento %d: %w", totalChunks, errEnvio)
			}
		}
		if err == io.EOF {
			fmt.Printf("[gRPC] Audio %d enviado completo (%d chunks)\n", audioId, totalChunks)
			return nil
		}
		if err != nil {
			return fmt.Errorf("error leyendo el archivo: %w", err)
		}
	}
}

// publicarReproduccion envía el evento a la cola. Un fallo aquí no aborta el
// streaming: la cola existe justamente para desacoplar ambos servidores.
func (f *FachadaStreaming) publicarReproduccion(audioId int, req *pb.AudioRequest) {
	if f.publicador == nil {
		return
	}
	mensaje := dto.ReproduccionDTO{
		AudioId:   audioId,
		Titulo:    req.Titulo,
		Tipo:      req.Tipo,
		Fecha:     time.Now().Format(time.RFC3339),
		UsuarioId: req.UsuarioId,
	}
	if err := f.publicador.PublicarReproduccion(mensaje); err != nil {
		fmt.Printf("[Cola] Error publicando reproducción: %v\n", err)
	}
}
