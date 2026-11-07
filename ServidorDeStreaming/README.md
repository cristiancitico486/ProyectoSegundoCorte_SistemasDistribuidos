# ServidorDeStreaming

gRPC (streaming del lado del servidor) + publicación asíncrona en RabbitMQ.
Capas: main -> controlador (gRPC) -> fachada -> clientes (REST hacia
ServidorDeAudios) / publicador (RabbitMQ).

## 1. Generar el código del .proto

Los archivos generados ya vienen incluidos. Si se modifica `streaming.proto`,
regenerarlos desde `ServidorDeStreaming/pb`:

```
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       streaming.proto
```

## 2. Dependencias y ejecución

```
cd ServidorDeStreaming
go mod tidy
go run .
```

Escucha en `:50051` (gRPC). Variables de entorno opcionales:
- `AUDIOS_URL` (por defecto `http://localhost:8082`)
- `RABBITMQ_URL` (por defecto `amqp://admin:1234@localhost:5672/`)

## Qué hace al recibir una petición

1. Imprime el eco del RPC recibido.
2. Pide el archivo a `GET /audios/{id}/archivo` en `ServidorDeAudios`; si no
   existe, devuelve error y no cuenta la reproducción.
3. Publica en la cola `reproducciones_audio` (mensaje JSON persistente). Si
   RabbitMQ no está disponible, solo avisa por consola, reintenta en la
   siguiente reproducción y no bloquea el streaming.
4. Reenvía el archivo en chunks de 32 KB por el stream gRPC, sin cargarlo
   completo en memoria.

## Segundo corte: usuarioId

`AudioRequest` gana `string usuario_id = 4;`. El servidor lo copia en el mensaje que publica en la cola `reproducciones_audio` (`usuarioId`). El servicio gRPC (`AudioService/AudioStream`) no cambia, por lo que `envoyConfig2.yaml` tampoco.
Para regenerar `pb/streaming.pb.go`: `protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative pb/streaming.proto` (desde `ServidorDeStreaming/`, ajusta la ruta si ejecutas dentro de `pb/`).
