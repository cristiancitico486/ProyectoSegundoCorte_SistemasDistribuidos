# Administrador

Programa de consola aparte del Cliente (ya no es un modo `--admin`). Sube
audios nuevos en dos pasos automáticos:

1. Registra título, tipo y metadatos en `ServidorMetadataDeAudios`
   (`POST /audios`) y obtiene el `id` generado.
2. Sube el `.mp3` local a `ServidorDeAudios` (`POST /audios/{id}`) usando
   ese mismo `id`.

Capas: main -> controlador -> fachada -> clientes (REST) / vista (consola)

## Requisitos antes de correrlo

Deben estar corriendo:
- `ServidorMetadataDeAudios` en `:8081`
- `ServidorDeAudios` en `:8082`

## Correr

```
cd Administrador
go run .
```

Si los servidores corren en otra máquina:

```
METADATOS_URL=http://192.168.1.10:8081 AUDIOS_URL=http://192.168.1.10:8082 go run .
```

## Flujo en consola

1. "Subir nuevo audio"
2. Elegir el tipo (Música, Podcasts, Audiolibros o Ruido Blanco)
3. Escribir el título
4. Ingresar metadatos como pares clave/valor (clave vacía para terminar)
5. Indicar la ruta local del `.mp3`
6. Confirmar el resumen

Si falla la subida del mp3 después de haber registrado los metadatos, el
mensaje de error incluye el `id` ya asignado para poder reintentar sin
duplicar el registro.
