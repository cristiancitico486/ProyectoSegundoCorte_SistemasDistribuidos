# ServidorDeAudios

REST puro con net/http (sin dependencias externas). Guarda y sirve los .mp3
en la carpeta local `audios/`. El nombre de archivo es siempre `{id}.mp3`,
usando el mismo id que asignó el ServidorMetadataDeAudios.

Capas: main -> controlador -> fachada -> repositorio

## Correr

```
cd ServidorDeAudios
go run .
```

Escucha en `:8082`.

## Probar con curl

Subir un mp3 para el audio con id=1 (debe existir antes en el
ServidorMetadataDeAudios):

```
curl -X POST http://localhost:8082/audios/1 \
  -F "archivo=@/ruta/a/tu/cancion.mp3"
```

Descargar/leer el archivo (esto es lo que hará el ServidorDeStreaming):

```
curl http://localhost:8082/audios/1/archivo --output prueba.mp3
```

## Validación

Solo se aceptan archivos mp3 (etiqueta ID3 o frame MPEG al inicio); cualquier
otro contenido responde 400.

## Nota de integración

Este servidor NO publica en RabbitMQ: esa responsabilidad es del
ServidorDeStreaming al reproducir. Aquí solo se almacena y se sirve el archivo.

Los 8 audios iniciales (`audios/1.mp3` a `audios/8.mp3`) son archivos de
prueba livianos, dos por cada tipo, que corresponden a los ids del catálogo.

## Segundo corte

- `GET /health` → `{"status":"ok"}` (lo consulta Consul).
- Errores: `{"error":"<codigo>","mensaje":"<texto>"}`. Códigos: `id_invalido` (el id debe ser entero positivo), `formulario_invalido`, `archivo_requerido`, `archivo_vacio`, `archivo_no_mp3`, `archivo_muy_grande` (413, máximo 200 MB), `audio_no_encontrado` (404), `error_interno` (500).
- Consul (opcional): con `CONSUL_ADDR=localhost:8500` se registra como `audios-service` (check HTTP a `/health`). `SERVICE_ADDR` define la dirección. Se desregistra al cerrar con Ctrl+C.
- Pruebas: `go test ./...`
