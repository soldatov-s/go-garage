#!/bin/bash
openssl req -newkey rsa:2048 -nodes -keyout server.key -out server.csr -config openssl.cnf
openssl x509 -req -days 3650 -in server.csr -signkey server.key -out server.crt -extensions v3_req -extfile openssl.cnf

chmod 664 server.key

openssl x509 -in server.crt -noout -text

