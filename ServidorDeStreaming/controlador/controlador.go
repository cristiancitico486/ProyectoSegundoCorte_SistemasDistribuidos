// Package controlador implementa el servidor gRPC (la C de MVC en este
// módulo). No contiene lógica de negocio: delega en la fachada.
package controlador

import (
	"streaming.local/servidor-streaming/fachada"
	"streaming.local/servidor-streaming/pb"
)

type ControladorStreaming struct {
	pb.UnimplementedAudioServiceServer
	fachada *fachada.FachadaStreaming
}

func NuevoControladorStreaming(f *fachada.FachadaStreaming) *ControladorStreaming {
	return &ControladorStreaming{fachada: f}
}

// AudioStream implementa el procedimiento remoto server-streaming del .proto.
func (c *ControladorStreaming) AudioStream(req *pb.AudioRequest, stream pb.AudioService_AudioStreamServer) error {
	return c.fachada.ReproducirAudio(req, stream)
}
