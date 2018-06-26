#!/bin/bash

SUBJECT="/C=US/ST=NY/L=Armonk/O=IBM/CN=dlaas.ibm.com"
VALIDITY=365 # days

echo Generate CA key:
openssl genrsa -des3  -passout pass:w3stw0r1d -out ca.key 4096

echo Generate CA certificate:
openssl req -new -x509 -days ${VALIDITY} -key ca.key -passin pass:w3stw0r1d -out ca.crt -subj ${SUBJECT}

echo Generate server key:
openssl genrsa -des3  -passout pass:w3stw0r1d -out server.key 4096

echo Generate server signing request:
openssl req -new -key server.key -passin pass:w3stw0r1d -out server.csr -subj ${SUBJECT}

echo Self-sign server certificate:
openssl x509 -req -days ${VALIDITY} -in server.csr -CA ca.crt -CAkey ca.key -set_serial 01 -passin pass:w3stw0r1d -out server.crt

echo Remove passphrase from the server key:
openssl rsa -in server.key -passin pass:w3stw0r1d -out server.key

echo Generate client key:
openssl genrsa -des3  -passout pass:w3stw0r1d -passout pass:w3stw0r1d -out client.key 4096

echo Generate client signing request:
openssl req -new -key client.key -passin pass:w3stw0r1d -out client.csr -subj ${SUBJECT}

echo Self-sign client certificate:
openssl x509 -req -days ${VALIDITY} -in client.csr -CA ca.crt -CAkey ca.key -set_serial 01 -passin pass:w3stw0r1d -out client.crt

echo Remove passphrase from the client key:
openssl rsa -in client.key -passin pass:w3stw0r1d -out client.key
