// cliente_receptor_fragmentos.js
// Recibe por gRPC-Web (a través de Envoy) los fragmentos de un audio, los
// junta y los entrega al reproductor. Se empaqueta con `npx webpack`.

const { AudioRequest } = require('./stubs_generados/streaming_pb.js');
const { AudioServicePromiseClient } = require('./stubs_generados/streaming_grpc_web_pb.js');

const ENVOY_HOST = (window.CONFIG && window.CONFIG.envoy) || 'http://localhost:8080';
const client = new AudioServicePromiseClient(ENVOY_HOST);

// Fragmentos del audio que se está recibiendo
let audioChunks = [];
let _firstFragmentReceived = false;
let _urlActual = null;

function eco(mensaje, nivel) {
    if (window.__writeAudioLog) {
        window.__writeAudioLog(mensaje, nivel);
    }
}

function iniciar_streaming_audio(audioId, titulo, tipo, usuarioId) {
    console.log('Iniciando llamada gRPC-Web a través de Envoy...');

    const audioPlayer = document.getElementById('audio-player');
    audioChunks = [];
    _firstFragmentReceived = false;

    const request = new AudioRequest();
    request.setAudioId(audioId);
    request.setTitulo(titulo);
    request.setTipo(tipo);
    request.setUsuarioId(usuarioId);
    console.log('Solicitud preparada:', request.toObject());

    const stream = client.audioStream(request, {});

    stream.on('data', (response) => {
        const data = response.getData_asU8();
        console.log(`Fragmento recibido desde el servidor. Tamaño: ${data.length} bytes`);
        audioChunks.push(data);

        if (!_firstFragmentReceived) {
            _firstFragmentReceived = true;
            if (audioPlayer) {
                audioPlayer.style.display = '';
            }
            eco('Primer fragmento recibido. Reproductor mostrado.', 'success');
        }
    });

    stream.on('end', () => {
        console.log('✅ Transmisión de la canción finalizada. Preparando reproducción...');

        const audioBlob = new Blob(audioChunks, { type: 'audio/mpeg' });
        if (_urlActual) {
            URL.revokeObjectURL(_urlActual);
        }
        _urlActual = URL.createObjectURL(audioBlob);
        audioChunks = [];
        _firstFragmentReceived = false;

        // Avisa a funciones.js qué audio quedó cargado (para el canal de reacciones)
        if (window.__marcarAudioCargado) {
            window.__marcarAudioCargado(audioId);
        }
        if (audioPlayer) {
            audioPlayer.src = _urlActual;
            audioPlayer.play()
                .then(() => {
                    console.log('▶️ Reproducción de la canción iniciada con éxito.');
                })
                .catch(error => {
                    console.error('❌ Error al iniciar la reproducción automática. Intente dar clic en el reproductor.', error);
                });
        }
    });

    stream.on('error', (err) => {
        console.error('ERROR en la comunicación gRPC-Web:', err);
        audioChunks = [];
        _firstFragmentReceived = false;
        eco(`Error en el streaming: ${err.message || err.code}`, 'error');
    });
}

if (typeof window !== 'undefined') {
    window.iniciar_streaming_audio = iniciar_streaming_audio;
    console.log('Exported iniciar_streaming_audio to window');
}
