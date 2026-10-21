package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"

	"drayya-core/internal/automigrate"
	"drayya-core/internal/database"
	"drayya-core/internal/grpcserver"
)

func main() {
	_ = godotenv.Load(".env")

	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	log.Printf("==================================================")
	log.Printf("  MrBiz-OS Pure gRPC / ConnectRPC Modular Engine  ")
	log.Printf("  Protocol: HTTP/2 H2C + Connect-Web + gRPC      ")
	log.Printf("  REST Status: COMPLETELY DECOMMISSIONED          ")
	log.Printf("==================================================")

	// Connect to Database
	database.Connect()

	// Run domain schema migrations
	automigrate.Run(database.DB)

	server := grpcserver.NewServer(database.DB)
	server.RegisterAllServices()

	log.Printf("Server listening on http://localhost%s", addr)
	if err := server.Start(addr); err != nil {
		log.Fatalf("gRPC Server terminated: %v", err)
	}
}
