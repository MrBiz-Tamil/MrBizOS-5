#!/bin/bash
set -e

echo "Generating Swagger Documentation..."
# We run this from the core-go directory
go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/server/main.go --parseDependency --parseInternal

echo "Swagger JSON generated at docs/swagger.json"
