# API Gateway

Punto único de entrada del cliente web y del administrador web (puerto **9000**). Reenvía cada petición al servicio que corresponde, descubriéndolo por Consul o por URLs fijas. Solo biblioteca estándar de Go (sin dependencias).

El streaming de audio **no** pasa por aquí: el cliente habla con Envoy (`:8080`).

## Rutas (contratos IA 1 v1.0)

| Ruta en el gateway | Servicio (Consul) | Ruta en el servicio |
|---|---|---|
| `GET /api/tipos` | metadata-service | `/tipos` |
| `GET /api/tipos/{id}/audios` | metadata-service | `/tipos/{id}/audios` |
| `GET /api/tipos/{id}/campos` | metadata-service | `/tipos/{id}/campos` |
| `GET /api/audios/{id}` | metadata-service | `/audios/{id}` |
| `POST /api/audios` | metadata-service | `/audios` |
| `POST /api/audios/{id}/archivo` | audios-service | `/audios/{id}` (multipart, campo `archivo`) |
| `GET /api/historial/{usuarioId}` | estadisticas-service | `/estadisticas/historial/{usuarioId}` |
| `GET /api/planes` | suscripciones-service | `/planes` |
| `POST /api/pagos` | suscripciones-service | `/pagos` |
| `GET /api/suscripciones/{usuarioId}` | suscripciones-service | `/suscripciones/{usuarioId}` |
| `WS /ws/reacciones?usuarioId=...` | reacciones-service | `/ws` (conserva el query string) |
| `GET /health` | (lo responde el gateway) | |

Cuerpo y código de estado del servicio pasan sin cambios. Las rutas de suscripciones responden error hasta que exista ese servicio (paso 5).

## Descubrimiento

- **Con `CONSUL_ADDR`** (por ejemplo `localhost:8500`): consulta `/v1/health/service/<nombre>?passing`, reparte entre instancias sanas (round-robin) y cachea 5 s. Si Consul no responde, usa la última respuesta conocida. Sin instancia sana: `503 servicio_no_disponible`.
- **Sin `CONSUL_ADDR`**: URLs fijas, cambiables por variable de entorno: `METADATA_URL` (`http://localhost:8081`), `AUDIOS_URL` (`:8082`), `REACCIONES_URL` (`:8083`), `ESTADISTICAS_URL` (`:8084`), `SUSCRIPCIONES_URL` (`:8085`). Si la URL no responde: `502 error_gateway`.

## Detalles

- **CORS:** `Access-Control-Allow-Origin: *` (cubre el Origin `null` de `file://`), métodos GET, POST, OPTIONS; el preflight se responde en cualquier ruta aceptando las cabeceras que anuncie el navegador. Las cabeceras CORS que traiga un servicio se descartan para que no se dupliquen.
- **Errores propios:** `{"error":"<codigo>","mensaje":"<texto>"}` — `ruta_no_encontrada` (404), `servicio_no_disponible` (503), `error_gateway` (502), `cuerpo_muy_grande` (413).
- **Subida de mp3:** se reenvía en flujo (el gateway no la carga en memoria), tope de 200 MB. Los POST JSON admiten hasta 1 MB.
- **Sin `ReadTimeout` ni `WriteTimeout`** en el servidor: cortarían los WebSocket (duran minutos) y la subida de mp3. Solo `ReadHeaderTimeout` de 10 s.
- **Ecos:** una línea por petición, por ejemplo `[Gateway] GET /api/tipos -> metadata-service (127.0.0.1:8081)`.
- Cierra limpio con Ctrl+C o SIGTERM.

## Ejecutar

```
go run .                                  # URLs fijas
CONSUL_ADDR=localhost:8500 go run .       # descubrimiento por Consul
go test ./...
```

Prueba rápida (con Metadata en :8081):

```
curl -s localhost:9000/health
curl -s localhost:9000/api/tipos
```
