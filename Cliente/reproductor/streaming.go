// StreamingGRPC es la implementación de Reproductor: llama al
// ServidorDeStreaming por gRPC, recibe el mp3 en chunks y lo reproduce en
// vivo mientras llegan, con un io.Pipe entre la recepción y la
// decodificación (github.com/faiface/beep + oto).
package reproductor

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"streaming.local/servidor-streaming/pb"

	"cliente.local/cliente-audio/vista"
)

// El speaker de beep se inicializa una sola vez por proceso; los audios
// siguientes se remuestrean a esa frecuencia si es distinta.
var (
	inicioSpeaker    sync.Once
	frecuenciaSalida beep.SampleRate
)

type StreamingGRPC struct {
	streamingURL string
}

// NuevoStreamingGRPC recibe la dirección del ServidorDeStreaming, por
// ejemplo "localhost:50051".
func NuevoStreamingGRPC(streamingURL string) *StreamingGRPC {
	return &StreamingGRPC{streamingURL: streamingURL}
}

// Reproducir pide el audio por gRPC, lo reproduce mientras llegan los
// chunks y muestra la pantalla "Reproduciendo audio" con la opción de
// salir en cualquier momento. Salir cancela el context, lo que corta el
// stream gRPC del lado del cliente.
func (r *StreamingGRPC) Reproducir(audioId int, titulo string, tipoNombre string) error {
	conn, err := grpc.NewClient(r.streamingURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("no se pudo conectar al servidor de streaming: %w", err)
	}
	defer conn.Close()

	client := pb.NewAudioServiceClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.AudioStream(ctx, &pb.AudioRequest{
		AudioId: int32(audioId),
		Titulo:  titulo,
		Tipo:    tipoNombre,
	})
	if err != nil {
		return fmt.Errorf("no se pudo iniciar el streaming: %w", err)
	}

	reader, writer := io.Pipe()
	errReproduccion := make(chan error, 1)
	audioTerminado := make(chan struct{})
	go decodificarYReproducir(reader, audioTerminado, errReproduccion)

	errRecepcion := make(chan error, 1)
	go recibirChunks(stream, writer, errRecepcion)

	// El teclado se vigila en una goroutine aparte; ver vigilarSalida.
	salidaUsuario := make(chan struct{})
	reproduccionCerrada := make(chan struct{})
	vigilanciaTerminada := make(chan struct{})
	go vigilarSalida(salidaUsuario, reproduccionCerrada, vigilanciaTerminada)

	vista.MostrarPantallaReproduccion(titulo)

	select {
	case <-salidaUsuario:
		cancel()
		writer.CloseWithError(io.EOF)
		speaker.Clear()
		fmt.Println("Reproducción interrumpida.")
		return nil
	case err := <-errRecepcion:
		if err != nil && err != io.EOF {
			return finalizarConError(reproduccionCerrada, vigilanciaTerminada, fmt.Errorf("error recibiendo el audio: %w", err))
		}
	}

	// Terminó de llegar el audio; se espera a que termine de sonar o a que
	// el usuario decida salir.
	select {
	case <-audioTerminado:
		select {
		case errDecodificacion := <-errReproduccion:
			if errDecodificacion != nil {
				return finalizarConError(reproduccionCerrada, vigilanciaTerminada, errDecodificacion)
			}
		default:
		}
		fmt.Println("Reproducción finalizada. Presione Enter para volver al menú.")
		close(reproduccionCerrada)
		<-vigilanciaTerminada
	case <-salidaUsuario:
		speaker.Clear()
		fmt.Println("Reproducción interrumpida.")
	}
	return nil
}

// finalizarConError libera la goroutine del teclado antes de devolver el
// error, para que no se quede con la siguiente línea que escriba el usuario.
func finalizarConError(reproduccionCerrada, vigilanciaTerminada chan struct{}, err error) error {
	fmt.Println("La reproducción terminó con un error. Presione Enter para continuar.")
	close(reproduccionCerrada)
	<-vigilanciaTerminada
	return err
}

func decodificarYReproducir(reader *io.PipeReader, audioTerminado chan struct{}, errCh chan error) {
	streamer, format, err := mp3.Decode(io.NopCloser(reader))
	if err != nil {
		errDecodificacion := fmt.Errorf("error decodificando mp3: %w", err)
		reader.CloseWithError(errDecodificacion)
		errCh <- errDecodificacion
		close(audioTerminado)
		return
	}

	inicioSpeaker.Do(func() {
		frecuenciaSalida = format.SampleRate
		speaker.Init(frecuenciaSalida, frecuenciaSalida.N(time.Second/2))
	})

	var fuente beep.Streamer = streamer
	if format.SampleRate != frecuenciaSalida {
		fuente = beep.Resample(4, format.SampleRate, frecuenciaSalida, streamer)
	}
	speaker.Play(beep.Seq(fuente, beep.Callback(func() {
		streamer.Close()
		close(audioTerminado)
	})))
}

func recibirChunks(stream pb.AudioService_AudioStreamClient, writer *io.PipeWriter, errCh chan error) {
	for {
		fragmento, err := stream.Recv()
		if err == io.EOF {
			writer.Close()
			errCh <- nil
			return
		}
		if err != nil {
			writer.CloseWithError(err)
			errCh <- err
			return
		}
		if _, err := writer.Write(fragmento.Data); err != nil {
			errCh <- err
			return
		}
	}
}

// vigilarSalida lee del teclado mientras suena el audio. Termina cuando el
// usuario elige "1. Salir" (cierra salidaUsuario) o cuando la reproducción
// ya se cerró por su cuenta (reproduccionCerrada): en ese caso la línea que
// llegue se consume aquí, por eso se le pide al usuario presionar Enter y
// no queda ninguna lectura pendiente compitiendo con el menú principal.
func vigilarSalida(salidaUsuario chan struct{}, reproduccionCerrada chan struct{}, terminada chan struct{}) {
	defer close(terminada)
	for {
		opcion := vista.LeerOpcion()
		select {
		case <-reproduccionCerrada:
			return
		default:
		}
		if opcion == 1 {
			close(salidaUsuario)
			return
		}
		vista.MostrarOpcionInvalida()
	}
}
