#!/bin/bash

export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=esun
export DB_PASSWORD=
export DB_NAME=investorcenter_db
export DB_SSLMODE=disable
export PORT=8080

./investorcenter-api
