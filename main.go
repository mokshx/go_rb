package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"gorb/config"
	"gorb/db"
	"gorb/rb"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("failed to load .env:", err)
	}

	conf, err := config.Load()
	if err != nil {
		log.Fatal("failed to load config:", err)
	}

	database, err := db.InitDB(conf)
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}
	defer database.Close()

	// Build the App with an optional S3 client.
	// Set S3_BUCKET in .env to enable attachment embedding.
	app := &rb.App{DB: database}
	if bucket := os.Getenv("S3_BUCKET"); bucket != "" {
		s3Client, err := rb.NewS3Client(context.Background(), bucket)
		if err != nil {
			log.Printf("S3 client init failed (attachments disabled): %v", err)
		} else {
			app.S3 = s3Client
		}
	}

	mux := http.NewServeMux()

	// POST /search-package
	// Body: { "orderId": "...", "builderId": "...", "adminUserId": "...", "isPreview": false }
	mux.HandleFunc("POST /search-package", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			OrderID     string `json:"orderId"`
			BuilderID   string `json:"builderId"`
			AdminUserID string `json:"adminUserId"`
			IsPreview   bool   `json:"isPreview"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		req := rb.CreateSearchPackageRequest{
			OrderID:     body.OrderID,
			BuilderID:   body.BuilderID,
			AdminUserID: body.AdminUserID,
			IsPreview:   body.IsPreview,
		}

		result, err := app.CreateSearchPackage(r.Context(), req)
		if err != nil {
			log.Printf("CreateSearchPackage error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(result); err != nil {
			log.Printf("encode response error: %v", err)
		}
	})

	log.Println("server listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal("server error:", err)
	}
}
