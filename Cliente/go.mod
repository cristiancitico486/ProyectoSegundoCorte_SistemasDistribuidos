module cliente.local/cliente-audio

go 1.24.5

require (
	github.com/faiface/beep v1.1.0
	google.golang.org/grpc v1.74.2
	streaming.local/servidor-streaming v0.0.0
)

require (
	github.com/hajimehoshi/go-mp3 v0.3.0 // indirect
	github.com/hajimehoshi/oto v0.7.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/exp v0.0.0-20190306152737-a1d7652674e8 // indirect
	golang.org/x/image v0.0.0-20190227222117-0694c2d4d067 // indirect
	golang.org/x/mobile v0.0.0-20190415191353-3e0bab5405d6 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250528174236-200df99c418a // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)

// El Cliente usa el paquete pb generado dentro de ServidorDeStreaming en
// vez de duplicar el .proto; deben quedar en carpetas hermanas al empaquetar
// (Cliente/ y ServidorDeStreaming/ al mismo nivel).
replace streaming.local/servidor-streaming => ../ServidorDeStreaming
