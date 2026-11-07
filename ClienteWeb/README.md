# ClienteWeb (paso 8)

Cliente web del segundo corte: catálogo, streaming gRPC-Web por Envoy, reacciones por WebSocket e historial.
Contratos: `contratos_IA2_v0.3.md`.

## Preparar (Debian)
    cd ProyectoPrimerCorte/ClienteWeb
    npm install
    bash generar_stubs.sh        # usa grpc_tools_node_protoc + protoc-gen-grpc-web local
    npx webpack                  # genera bundle.js
    # abrir index.html en el navegador

Nota: el paquete `protobuf-compiler` de Debian trae `protoc`, pero no trae el plugin `protoc-gen-js` necesario para `--js_out`. Por eso el script usa las herramientas locales del proyecto (`grpc_tools_node_protoc` + `protoc-gen-grpc-web`) y no depende de un binario global.

## Configuración (en la URL)
    index.html?api=http://localhost:9000/api&ws=ws://localhost:9000/ws/reacciones&envoy=http://localhost:8080
Valores por defecto: los de arriba. Para probar el canal de reacciones sin gateway: `index.html?ws=ws://localhost:8083/ws`.

## Qué usa
- Catálogo: `GET {api}/tipos`, `/tipos/{id}/audios`, `/audios/{id}` (respuestas como las de ServidorMetadataDeAudios).
- Historial: `GET {api}/historial/{usuarioId}`.
- Streaming: `AudioStream(audio_id, titulo, tipo, usuario_id)` por Envoy :8080.
- Reacciones: `registrar` en `play`, `desregistrar` en `pause`/fin (funciones.js).
- `usuarioId` se genera una vez y se guarda en localStorage.
