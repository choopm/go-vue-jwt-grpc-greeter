#!/bin/sh
set -e

if ! openssl x509 -text -noout -in $TLS_CRT | grep DNS >/dev/null || [ ! -f $TLS_CRT ]; then
  echo "Missing $TLS_CRT"
  echo "generating one"
  rm -rf $TLS_KEY
  rm -rf $TLS_CRT

  openssl req \
    -x509 \
    -nodes \
    -days 3650 \
    -sha256 \
    -newkey rsa:2048 \
    -keyout $TLS_KEY \
    -out $TLS_CRT \
    -subj "/C=DE/O=0pointer.org/CN=app" \
    -addext "subjectAltName = DNS:app,IP:0.0.0.0"
else
  echo "Using certificate $TLS_CRT:"
  openssl x509 -text -nocert -in $TLS_CRT
fi

if [ ! -f $JWT_SECRET ]; then
  echo "Missing $JWT_SECRET"
  echo "generating one"
  openssl rand -base64 32 > $JWT_SECRET
fi

echo "Exec $@"
exec $@
