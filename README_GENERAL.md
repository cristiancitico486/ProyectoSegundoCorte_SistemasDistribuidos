# ORden de ejecucion
docker run -d --name rabbitmq -p 5672:5672 -p 15672:15672 -e RABBITMQ_DEFAULT_USER=admin -e RABBITMQ_DEFAULT_PASS=1234 rabbitmq:3-management

docker start (id del anterior(abajo puse uno ya creado en mi pc))


cd ProyectoPrimerCorte/ServidorMetadataDeAudios && go mod tidy && go run .

cd ProyectoPrimerCorte/ServidorDeAudios && go mod tidy && go run .

cd ProyectoPrimerCorte/ServidorDeStreaming && go mod tidy && go run .

cd ProyectoPrimerCorte/ServidorDeEstadisticas && go mod tidy && go run .

cd ProyectoPrimerCorte/Cliente && go mod tidy && go run .

EL docker que estoy usando es el 119e6aed43a1 con admin/1234