package rb_test

import (
	"context"
	"encoding/base64"
	"gorb/config"
	"gorb/db"
	"gorb/rb"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestCreateSearchPackage(t *testing.T) {
	// Load environment variables from .env
	// Try loading from parent directory (if run from rb/) or current directory
	if err := godotenv.Load("../.env"); err != nil {
		if err := godotenv.Load(".env"); err != nil {
			t.Log("warning: no .env file found, relying on environment variables")
		}
	}

	conf, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	database, err := db.InitDB(conf)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer database.Close()

	app := &rb.App{DB: database}




	req := rb.CreateSearchPackageRequest{
		OrderID:     "384788",
		BuilderID:   "7574",
		AdminUserID: "4c96fd2b-ba23-4451-b1be-f55fac496e87",
	}
	res, err := app.CreateSearchPackage(context.Background(), req)
	if err != nil {

		t.Logf("CreateSearchPackage returned error (as expected/possible): %v", err)
	} else {
		t.Logf("CreateSearchPackage succeeded!")
		if res != nil {
			var pdfData []byte
			if res.IsPreview {
				pdfData = res.PDFBuffer
			} else {
				var err error
				pdfData, err = base64.StdEncoding.DecodeString(res.UpdatePDFBase64)
				if err != nil {
					t.Fatalf("failed to decode base64 PDF: %v", err)
				}
			}
			t.Logf("IsPreview: %v, UpdatePDFBase64 length: %d, Decoded PDF length: %d",
				res.IsPreview, len(res.UpdatePDFBase64), len(pdfData))

			// Save the generated PDF
			outputPath := "../rb_test_output.pdf"
			if err := os.WriteFile(outputPath, pdfData, 0644); err != nil {
				t.Logf("failed to write output PDF file: %v", err)
			} else {
				t.Logf("Successfully wrote generated PDF to %s", outputPath)
			}
		}
	}
}
