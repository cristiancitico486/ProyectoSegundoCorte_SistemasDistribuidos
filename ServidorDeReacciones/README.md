# ServidorDeReacciones (WebSocket, :8083)

Contrato: `contratos_IA2_v0.3.md`, sección 4. Solo biblioteca estándar de Go (sin dependencias).

## Ejecutar (Debian)
    go run .                                  # escucha en :8083
    CONSUL_ADDR=localhost:8500 go run .       # además se registra como reacciones-service
    (en Docker sin --network host añade SERVICE_ADDR=<IP de la máquina>)

## Probar
    go vet ./... && go test -race ./...
    # a mano, con websocat si lo tienes:  websocat "ws://localhost:8083/ws?usuarioId=ana"
    # y escribir: {"tipo":"registrar","audioId":1}
