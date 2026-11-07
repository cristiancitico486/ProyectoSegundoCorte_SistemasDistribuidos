# ServidorMetadataDeAudios

REST puro con net/http (sin dependencias externas). Capas:
main -> controlador -> fachada -> repositorio -> modelo/dto

## Correr

```
cd ServidorMetadataDeAudios
go run .
```

Escucha en `:8081`. El catálogo se persiste en `datos/catalogo.json`.

## Probar con curl

```
curl http://localhost:8081/tipos
curl http://localhost:8081/tipos/3/audios
curl http://localhost:8081/audios/5
curl -X POST http://localhost:8081/audios \
  -H "Content-Type: application/json" \
  -d '{"tipoId":1,"titulo":"Nueva Cancion","metadatos":{"Artista Principal":"X"}}'
```

## Segundo corte

- `GET /health` → `{"status":"ok"}` (lo consulta Consul).
- `GET /tipos/{id}/campos` → metadatos del tipo, obligatorios primero, con sugerencias de valores:
  `{"tipoId":1,"tipo":"Música","campos":[{"clave":"Artista Principal","obligatorio":true},...]}`.
  Los obligatorios son los del enunciado: Música (Artista Principal, Álbum, Género Musical), Podcasts (Nombre del Podcast, Título del Episodio, Anfitrión), Audiolibros (Título del Libro, Autor, Narrador), Ruido Blanco (Tipo de Sonido, Fuente del Audio, Uso Sugerido). Se definen en `modelo/campos.go`.
- `POST /audios` ahora valida: tipo existente, título, y los metadatos obligatorios del tipo. Acepta además metadatos libres (opcionales). Recorta espacios y descarta los vacíos.
- Errores: `{"error":"<codigo>","mensaje":"<texto>","faltantes":[...]}` (`faltantes` solo en `metadatos_invalidos`). Códigos: `id_invalido`, `cuerpo_invalido`, `tipo_invalido`, `tipo_no_encontrado`, `audio_no_encontrado`, `titulo_obligatorio`, `titulo_invalido`, `metadatos_invalidos`, `error_interno`.
- Consul (opcional): con `CONSUL_ADDR=localhost:8500` se registra como `metadata-service` (check HTTP a `/health`). `SERVICE_ADDR` define la dirección. Se desregistra al cerrar con Ctrl+C.
- Pruebas: `go test ./...`

### Ajustar el catálogo existente (una sola vez)

Detén el servidor y ejecuta:

```
go run ./herramientas/normalizar            # simulación: muestra qué cambiaría
go run ./herramientas/normalizar -aplicar   # guarda copia .bak y aplica
```

Renombra claves antiguas de Música a las del enunciado, agrega "Título del Episodio" a los podcasts que no lo tienen y lista los audios que siguen sin metadatos obligatorios (no inventa valores ni borra audios).
