# ServidorDeEstadisticas

Consume la cola `reproducciones_audio` de RabbitMQ (donde publica el
ServidorDeStreaming), almacena cada reproducción en `datos/reproducciones.jsonl`
y la imprime por pantalla junto con los contadores acumulados por tipo y por
audio. Cada mensaje consumido imprime un eco `[Cola] Mensaje consumido...`.

Capas: main -> controlador -> fachada -> repositorio -> modelo/dto;
consumidor (RabbitMQ) y vista (consola).

## Correr

```
cd ServidorDeEstadisticas
go mod tidy
go run .
```

Variable de entorno opcional: `RABBITMQ_URL` (por defecto
`amqp://admin:1234@localhost:5672/`). Si RabbitMQ aún no está arriba, el
servidor reintenta la conexión cada 3 segundos.

## Segundo corte: historial por usuario (REST :8084)

- Cada mensaje de la cola trae `usuarioId`. Las líneas antiguas de `datos/reproducciones.jsonl` (sin usuario) cuentan en el resumen global pero no en historiales.
- Al iniciar se relee `datos/reproducciones.jsonl` y se reconstruyen contadores e historial, así no se pierden al reiniciar.
- `GET /estadisticas/historial/{usuarioId}` → `{"usuarioId","totalReproducciones","audios":[{"audioId","titulo","tipo","cantidad","ultimaReproduccion"}]}`, con la reproducción más reciente primero. Usuario sin historial: 200 con `audios: []`.
- `GET /health` → `{"status":"ok"}`. Errores: `{"error":"<codigo>","mensaje":"<texto>"}`.
- Pruebas: `go test ./...`
