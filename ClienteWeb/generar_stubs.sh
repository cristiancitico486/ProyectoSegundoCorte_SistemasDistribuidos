#!/usr/bin/env bash
# Genera los stubs grpc-web (commonjs) a partir de streaming.proto.
# Usa los binarios locales del proyecto para evitar depender del plugin global
# `protoc-gen-js`, que no existe en Debian con el paquete `protobuf-compiler`.
set -e
cd "$(dirname "$0")"

if [ ! -d node_modules ]; then
  echo "Faltan dependencias del proyecto. Ejecuta: npm install"
  exit 1
fi

export PATH="$PWD/node_modules/.bin:$PATH"

mkdir -p stubs_generados
npx grpc_tools_node_protoc \
  -I. \
  --js_out=import_style=commonjs:stubs_generados \
  --grpc-web_out=import_style=commonjs,mode=grpcwebtext:stubs_generados \
  streaming.proto

echo "Stubs generados en stubs_generados/: $(ls stubs_generados | tr '\n' ' ')"
