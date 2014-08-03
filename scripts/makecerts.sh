#!/bin/bash
# call this script with an email address (valid or not).
# like:
# ./makecert.sh foo@foo.com
mkdir certs
rm certs/*
echo "make boogie server cert"
openssl req -new -nodes -x509 -out certs/server.pem -keyout certs/server.key -days 3650 -subj "/C=DE/ST=NRW/L=Earth/O=Example Company/OU=IT/CN=www.example.com/emailAddress=$1"
echo "make boogie client cert"
openssl req -new -nodes -x509 -out certs/client.pem -keyout certs/client.key -days 3650 -subj "/C=DE/ST=NRW/L=Earth/O=Example Company/OU=IT/CN=www.example.com/emailAddress=$1"

echo "make bagent server cert"
openssl req -new -nodes -x509 -out certs/bagent-server.pem -keyout certs/bagent-server.key -days 3650 -subj "/C=DE/ST=NRW/L=Earth/O=Example Company/OU=IT/CN=www.example.com/emailAddress=$2"
echo "make bagent client cert"
openssl req -new -nodes -x509 -out certs/bagent-client.pem -keyout certs/bagent-client.key -days 3650 -subj "/C=DE/ST=NRW/L=Earth/O=Example Company/OU=IT/CN=www.example.com/emailAddress=$2"

echo "make bcli server cert"
openssl req -new -nodes -x509 -out certs/bcli-server.pem -keyout certs/bcli-server.key -days 3650 -subj "/C=DE/ST=NRW/L=Earth/O=Example Company/OU=IT/CN=www.example.com/emailAddress=$3"
echo "make bcli client cert"
openssl req -new -nodes -x509 -out certs/bcli-client.pem -keyout certs/bcli-client.key -days 3650 -subj "/C=DE/ST=NRW/L=Earth/O=Example Company/OU=IT/CN=www.example.com/emailAddress=$3"
