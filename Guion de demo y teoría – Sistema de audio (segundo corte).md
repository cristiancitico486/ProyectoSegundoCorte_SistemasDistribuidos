# Guion de demo y teoría – Sistema de audio (segundo corte)

Nov 7, 2026 · @Cristian

## 1. Preparación: orden de arranque

Cada servicio va en su propia terminal, dentro de `ProyectoPrimerCorte/`, y se levanta en este orden; la última columna es la línea que confirma que está listo.

| # | Componente | Carpeta y comando | Puerto | Listo cuando |
| --- | --- | --- | --- | --- |
| 1 | RabbitMQ (Docker) | ya levantado | 5672 | el contenedor está activo |
| 2 | Consul (opcional) | como lo tengas en Docker | 8500 | la interfaz web abre |
| 3 | Metadata | `ServidorMetadataDeAudios`: `go run .` | 8081 | imprime que escucha |
| 4 | Audios | `ServidorDeAudios`: `go run .` | 8082 | imprime que escucha |
| 5 | Estadísticas | `ServidorDeEstadisticas`: `go run .` | 8084 | `[Cola] Conectado a RabbitMQ` |
| 6 | Reacciones | `ServidorDeReacciones`: `go run .` | 8083 | `escuchando en :8083` |
| 7 | Streaming | `ServidorDeStreaming`: `go run .` | 50051 | `escuchando en :50051` |
| 8 | API Gateway | `ApiGateway`: `go run .` | 9000 | imprime que escucha |
| 9 | Envoy | carpeta del parcial: `./envoyServer -c envoyConfig2.yaml --disable-hot-restart` | 8080 | `starting main dispatch loop` |
| 10 | Cliente web | abrir `ClienteWeb/index.html` como archivo | n/a | `Canal de reacciones: conectado` |

Con Consul, antepón `CONSUL_ADDR=localhost:8500` a cada `go run .` (de los pasos 3 al 8) y comprueba que todos aparezcan en verde. Si Consul corre en Docker sin `--network host`, añade también `SERVICE_ADDR=<IP de tu máquina>`.

Abre el cliente como archivo (`file:///...`), no con Live Server: Live Server recarga la página cuando Estadísticas escribe en `datos/reproducciones.jsonl` y corta el streaming.

El servidor de suscripciones (`:8085`) se suma a esta lista cuando IA 1 lo entregue.

## 2. Guion de demo por requerimiento

Hoy se pueden demostrar (a), (b) y (d), que ya verificó Cristian; (c), (e) y (f) dependen de los pasos 5 y 9 de IA 1 y aquí están escritos según su contrato v1.0.

### (b) Catálogo y streaming (listo)

1. Abrir `index.html`. Pulsar un tipo de audio (por ejemplo Música): aparece la lista de ese tipo.
2. Elegir un audio: se muestra su detalle, con los tres metadatos que pide el enunciado para ese tipo primero.
3. Pulsar Reproducir. En el log de ecos aparece `Primer fragmento recibido. Reproductor mostrado.` y el audio suena.

### (a) Reacciones por WebSocket (listo)

1. Abrir `index.html` en dos pestañas: cada una genera su propio usuario (se ve arriba).
2. Reproducir el mismo audio en las dos. Los ecos muestran `Registrado en el canal del audio N (oyentes: 1)` y luego `(oyentes: 2)`.
3. Pulsar una reacción en una pestaña: aparece en las dos, y la consola de Reacciones imprime `reenviada a 2 oyentes`.
4. Pausar en una pestaña: sus botones se desactivan y la consola imprime `desregistrado`.
5. En una tercera pestaña, reproducir otro audio y reaccionar: la reacción no llega a quienes escuchan el primero.

### (d) Historial (listo)

1. Reproducir el mismo audio dos veces.
2. Pulsar Ver mis audios reproducidos: la tabla muestra ese audio con 2 veces.
3. Detener y volver a iniciar Estadísticas y repetir la consulta: el conteo se conserva, porque se reconstruye desde el archivo.

### (c) Suscripción y pago (pendiente: IA 1, pasos 5 y 9)

1. Elegir un plan y pulsar Pagar. Ecos del cliente: `Token generado`, `Enviando pago`, `Pago exitoso`, `Pago almacenado por el servidor`. Consola del servidor: `Token NUEVO`, `Pago APROBADO`, `Pago ALMACENADO`.
2. Marcar simular respuesta lenta y pagar: el cliente espera 2000 ms, reenvía con el mismo token y el servidor imprime que el reenvío llega con el pago en curso. Solo hay un cobro.

### (e) Token repetido (pendiente: IA 1, pasos 5 y 9)

1. Tras un pago exitoso, pulsar Simular token repetido. El cliente imprime que el servidor ya había usado el token y devolvió el resultado original; el servidor imprime `Token YA UTILIZADO`. No se cobra de nuevo.

### (f) Administrador web (pendiente: IA 1, paso 9)

1. Abrir `admin.html`, elegir un tipo: el formulario pide los metadatos de ese tipo (tres obligatorios) y el archivo mp3.
2. Registrar el audio y comprobar que aparece en el catálogo del cliente y que se puede reproducir.

## 3. Ecos esperados en cada consola

El enunciado pide ecos en el cliente y en los servidores; estos son los que debe ver el evaluador durante la demo.

| Consola | Eco | Cuándo |
| --- | --- | --- |
| Log de la página | `Conectado al canal de reacciones.` | al abrir la página |
| Log de la página | `Primer fragmento recibido. Reproductor mostrado.` | llega el primer fragmento |
| Log de la página | `Reproducción iniciada (play)` / `Reproducción pausada (pause)` | al reproducir o pausar |
| Log de la página | `Registrado en el canal del audio N (oyentes: k).` | tras `play` |
| Consola del navegador (F12) | `Iniciando llamada gRPC-Web a través de Envoy...` y `Solicitud preparada` | al pedir el audio |
| Consola del navegador (F12) | `Fragmento recibido desde el servidor. Tamaño: N bytes` | por cada fragmento |
| Consola del navegador (F12) | `Transmisión de la canción finalizada. Preparando reproducción...` | llegan todos los fragmentos |
| Envoy | `all clusters initialized` y `starting main dispatch loop` | al arrancar |
| Streaming | `[gRPC] AudioStream llamado audioId=... usuarioId=...` | al pedir un audio |
| Streaming | `[Cola] Mensaje publicado en "reproducciones_audio"` | al iniciar el stream |
| Streaming | `[gRPC] Audio N enviado completo (k chunks)` | al terminar el envío |
| Estadísticas | `[Cola] Mensaje consumido de "reproducciones_audio"` | por cada reproducción |
| Estadísticas | `[REST] GET /estadisticas/historial/<usuario>` | al ver el historial |
| Reacciones | `[WS] ... registrado en audio N (oyentes: k)` | tras `play` |
| Reacciones | `[WS] Reacción "fuego" de ... reenviada a k oyentes` | al reaccionar |
| API Gateway | `[Gateway] GET /api/tipos -> metadata-service (127.0.0.1:8081)` | por cada petición |
| Cualquier servicio | `[Consul] Servicio "..." registrado en ...` | con `CONSUL_ADDR` definida |
| Suscripciones (pendiente) | `[Pago] Token NUEVO`, `Pago APROBADO`, `Pago ALMACENADO`, `Token YA UTILIZADO` | al pagar y al repetir el token |

## 4. Teoría de sistemas distribuidos y dónde está en el código

Doce conceptos sostienen el proyecto; para cada uno está la idea teórica, el archivo donde se implementa y cómo mostrarlo. Las rutas son relativas a `ProyectoPrimerCorte/`.

### 1. Microservicios y API Gateway

- **Teoría:** cada servicio hace una sola cosa y se despliega por separado; el gateway es el punto de entrada único, oculta dónde viven los servicios y centraliza CORS. El streaming no pasa por él: necesita HTTP/2 gRPC y por eso va por Envoy.
- **En el código:** `ApiGateway/` (módulo de IA 1) con la tabla de rutas `/api/...`; el cliente solo conoce `:9000` y `:8080` (`ClienteWeb/index.html`, objeto `CONFIG`).
- **En la demo:** `curl localhost:9000/api/tipos` y el eco `[Gateway] GET /api/tipos -> metadata-service`.

### 2. Descubrimiento de servicios (Consul)

- **Teoría:** cada servicio se registra con un nombre y un health check; los demás lo buscan por nombre, no por IP y puerto, y solo se usan instancias sanas.
- **En el código:** `consul/consul.go` en Streaming, Estadísticas y Reacciones (registro por la API HTTP, check TCP o HTTP, reintento cada 3 s, baja automática a 1 minuto de estar caído); el gateway consulta `?passing`, reparte en round-robin y cachea 5 s (IA 1).
- **En la demo:** la interfaz de Consul con los servicios en verde; detener uno y verlo pasar a rojo.

### 3. gRPC, HTTP/2 y gRPC-Web con Envoy

- **Teoría:** gRPC es RPC sobre HTTP/2 con Protocol Buffers y soporta streaming del servidor. El navegador no puede hablar gRPC nativo, así que usa gRPC-Web sobre HTTP/1.1 y Envoy traduce entre ambos.
- **En el código:** `ServidorDeStreaming/pb/streaming.proto` (`rpc AudioStream` devuelve `stream AudioChunk`); `fachada/fachada.go` envía con `stream.Send`; `ClienteWeb/cliente_receptor_fragmentos.js` recibe con `stream.on('data')`; `envoyConfig2.yaml` (filtros `cors` y `grpc_web`).
- **En la demo:** en la consola F12, los ecos `Fragmento recibido desde el servidor. Tamaño: N bytes`.

### 4. Protocol Buffers y evolución compatible del contrato

- **Teoría:** cada campo se identifica por su número, no por su nombre; añadir un campo con número nuevo no rompe a los clientes antiguos.
- **En el código:** `string usuario_id = 4;` en `AudioRequest` (`streaming.proto`); `envoyConfig2.yaml` no cambió porque package y servicio siguen igual.
- **En la demo:** mostrar el `.proto` y los stubs regenerados con `generar_stubs.sh`.

### 5. Streaming por fragmentos

- **Teoría:** enviar el archivo en trozos evita cargarlo entero en memoria. Este cliente junta todos los fragmentos y reproduce al terminar (no es progresivo), como el ejemplo del enunciado.
- **En el código:** `ServidorDeStreaming/fachada/fachada.go` (`tamanioChunk` = 32 KB, lee del servidor de Audios sin cargar el archivo completo); `cliente_receptor_fragmentos.js` arma el `Blob` en el evento `end`.
- **En la demo:** los ecos de fragmentos y el audio sonando al terminar la transmisión.

### 6. Comunicación asíncrona y colas de mensajes (RabbitMQ)

- **Teoría:** productor y consumidor quedan desacoplados en el tiempo: la cola recuerda el mensaje aunque el consumidor esté caído. Con cola durable, mensajes persistentes y confirmación manual (ack), cada mensaje se entrega al menos una vez.
- **En el código:** `ServidorDeStreaming/publicador/publicador.go` (cola durable, `DeliveryMode: Persistent`); `ServidorDeEstadisticas/consumidor/consumidor.go` (ack manual, `Qos` de 1, reconexión).
- **En la demo (sugerida, comprobarla antes):** detener Estadísticas, reproducir un audio, volver a iniciarla y ver que consume el mensaje pendiente.

### 7. Tolerancia a fallos y desacoplamiento

- **Teoría:** la caída de un componente no debe tumbar al resto; se reintenta en lugar de fallar.
- **En el código:** el streaming sigue funcionando sin RabbitMQ y reintenta al publicar (`ServidorDeStreaming/main.go`); el consumidor y el registro en Consul reintentan cada 3 s; el cliente reconecta el WebSocket cada 3 s (`funciones.js`); el gateway usa la última respuesta de Consul si este cae (IA 1).
- **En la demo (sugerida, comprobarla antes):** apagar RabbitMQ y comprobar que el audio sigue sonando.

### 8. WebSocket y notificación por callback

- **Teoría:** una conexión persistente y bidireccional que nace de un handshake HTTP Upgrade; el servidor puede enviar al cliente sin que este pregunte (callback), reutilizando la conexión ya abierta.
- **En el código:** `ServidorDeReacciones/ws/ws.go` (handshake: `Sec-WebSocket-Key` + GUID, SHA-1, base64 = `Sec-WebSocket-Accept`; frames con máscara; ping cada 30 s); `repositorio/registro.go` (mapa audio a clientes registrados, un canal de publicación y suscripción en memoria); en el cliente, `funciones.js` registra en `play` y desregistra en `pause`.
- **En la demo:** dos pestañas con el mismo audio y una reacción que llega a las dos.

### 9. Concurrencia

- **Teoría:** varias conexiones y peticiones tocan el mismo estado a la vez; se protege con exclusión mutua para evitar carreras de datos.
- **En el código:** `sync.Mutex` en `ServidorDeReacciones/repositorio/registro.go` y en el repositorio de Estadísticas; goroutines para el ping, y para el REST en paralelo al consumidor de la cola (`ServidorDeEstadisticas/main.go`); escritura serializada por conexión (`wmu` en `ws.go`).
- **En la demo:** `go test -race ./...` en Estadísticas y Reacciones.

### 10. Persistencia y reconstrucción del estado

- **Teoría:** guardar cada evento en un archivo de solo añadir y reconstruir el estado releyendo ese registro al arrancar.
- **En el código:** `ServidorDeEstadisticas/datos/reproducciones.jsonl` (una línea por evento) y `cargarExistentes()` en `repositorio/repositorio.go`, que rehace contadores e historial y acepta las líneas antiguas sin `usuarioId`.
- **En la demo:** reiniciar Estadísticas y repetir la consulta del historial.

### 11. Idempotencia, tokens y reenvíos (requerimientos c y e; IA 1, pendiente de entrega)

- **Teoría:** en una red no confiable el cliente no sabe si su mensaje llegó, así que reenvía; para que reenviar no cobre dos veces, el servidor debe ser idempotente: el mismo token produce el mismo efecto una sola vez.
- **En el código (según el contrato v1.0):** `POST /pagos` con `token` generado por el cliente; 201 si es nuevo, 200 con `repetido: true` si ya se usó, 409 si el token es de otro usuario o plan; los tokens se reconstruyen desde `pagos.jsonl`; `suscripciones.js` reenvía con el mismo token tras el temporizador.
- **En la demo:** pago normal, pago con respuesta lenta y botón Simular token repetido.

### 12. CORS, origen `null` y capas

- **Teoría:** el navegador bloquea peticiones a otro origen salvo que el servidor lo autorice; abrir la página como archivo envía `Origin: null`. Aparte, cada servicio separa controlador, fachada y acceso a datos para aislar la presentación, la lógica y la persistencia.
- **En el código:** bloque `cors` de `envoyConfig2.yaml` y `Access-Control-Allow-Origin: *` en el gateway (IA 1); carpetas `controlador`, `fachada` y `repositorio` en cada módulo.
- **En la demo:** abrir `index.html` como archivo y ver que el catálogo carga sin errores de CORS.

## 5. Preguntas probables de sustentación

Las respuestas dicen también las limitaciones reales del sistema; es mejor nombrarlas antes de que las pregunten.

- **¿Por qué el streaming no pasa por el gateway?** El cliente pide el audio directo a Envoy en `:8080`, como en el ejemplo del enunciado, porque Envoy ya traduce gRPC-Web a gRPC; el gateway atiende REST y WebSocket. El diagrama del enunciado dibuja Envoy detrás del gateway; aquí se prefirió no añadir un salto que no aporta nada.
- **¿Qué pasa si RabbitMQ se cae?** El streaming sigue funcionando, pero la reproducción de ese momento no se publica y no queda en el historial (se imprime el error y no se reintenta ese mensaje). Los mensajes que ya estaban en la cola sobreviven porque es durable y persistente.
- **¿La cola garantiza entrega exactamente una vez?** No: garantiza al menos una vez. Si Estadísticas se cae entre guardar la línea y confirmar el mensaje, ese mensaje podría contarse dos veces. Es una limitación aceptada en este laboratorio.
- **¿Por qué se publica al iniciar el stream y no al terminarlo?** Es el comportamiento heredado del primer corte, y solo se publica si el archivo existe. Contra: una reproducción cortada a mitad igual cuenta. Publicar al terminar es un cambio pequeño en `fachada.go`.
- **¿Por qué WebSocket y no consultar cada pocos segundos (polling)?** El servidor puede avisar en el instante en que llega una reacción, con una sola conexión abierta y sin peticiones vacías.
- **¿Qué pasa si Reacciones se reinicia?** El estado está en memoria, así que se pierden los registros; el cliente reconecta cada 3 s y, si sigue reproduciendo, se vuelve a registrar solo.
- **¿El sistema escala?** Estadísticas y los servicios REST sí, detrás del round-robin del gateway. Reacciones no sin cambios: con varias instancias, dos oyentes del mismo audio podrían caer en instancias distintas y no ver sus reacciones; haría falta un bus compartido.
- **¿Cómo se evita cobrar dos veces?** Con un token idempotente generado por el cliente: si el servidor ya lo usó, devuelve el resultado original sin cobrar (requerimientos c y e, pendiente de entrega).
- **¿Hay autenticación?** No: el enunciado no la pide. El `usuarioId` lo genera el navegador y cualquiera podría suplantarlo; es una limitación conocida.

## 6. Estado y pendientes

El sistema funciona de punta a punta salvo suscripción, pago y administrador web, que esperan a IA 1.

| Paso | Qué es | Responsable | Estado |
| --- | --- | --- | --- |
| 1 | Metadata y Audios con `/health`, Consul y campos por tipo | IA 1 | verificado |
| 2 | `.proto` con `usuario_id` y Streaming con Consul | IA 2 | verificado |
| 3 | API Gateway y proxy REST | IA 1 | verificado |
| 4 | Estadísticas con historial por usuario | IA 2 | verificado |
| 5 | Servidor de suscripciones (`:8085`) | IA 1 | pendiente |
| 6 | Servidor de reacciones (`:8083`) | IA 2 | verificado |
| 7 | Historial y WebSocket por el gateway | IA 1 | verificado |
| 8 | Cliente web | IA 2 | verificado |
| 9 | `suscripciones.js` y `admin.html` | IA 1 | pendiente |
| 10 | Integración de suscripciones y prueba final | IA 2 | espera los pasos 5 y 9 |

Sin definir: la fecha de entrega y el formato de la sustentación. Cuando estén pendientes (c), (e) y (f), actualizar las secciones 2 y 3 con los ecos reales que imprima el servidor de suscripciones.
