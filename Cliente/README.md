# Cliente

Cliente de consola (menús). Capas: main -> controlador -> fachada -> clientes
(REST) / reproductor (gRPC) -> vista (consola).

Consulta tipos, audios y detalles por REST al `ServidorMetadataDeAudios` y
reproduce por gRPC contra el `ServidorDeStreaming`, decodificando el mp3
mientras llegan los chunks (`beep` + `oto`).

## Requisitos antes de correrlo

Deben estar corriendo:
- `ServidorMetadataDeAudios` en `:8081`
- `ServidorDeAudios` en `:8082`
- `ServidorDeStreaming` en `:50051` (para reproducir)

`ServidorDeStreaming` debe ser carpeta hermana de `Cliente` (mismo nivel): el
`go.mod` del Cliente usa un `replace` hacia `../ServidorDeStreaming` para
reutilizar el paquete `pb` sin duplicar el `.proto`.

## Dependencia del sistema para sonido

```
sudo apt install -y libasound2-dev
```

## Correr

```
cd Cliente
go mod tidy
go run .
```

Variables de entorno opcionales: `METADATOS_URL`, `STREAMING_URL`.
