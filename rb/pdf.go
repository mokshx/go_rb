// Package rb — PDF generation pipeline.
//
// Mirrors the following functions from pdf_helper.js:
//   generateHTMLTemplate  → generateHTMLTemplate (entry point)
//   printPdf              → printPDF (chromedp headless Chrome)
//   generateCoverPage     → generateCoverPage
//   mergePDFDocuments     → mergePDFs (pdfcpu)
//   getBufferFromS3Bucket → downloadAttachments
//   getCurrentOwnersList  → getCurrentOwnersList
//   getNamesSearchedList  → getNamesSearchedList
package rb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/gpdf-dev/gpdf"
)

const (
	// Google Maps static API key (same as the JS project)
	googleMapsAPIKey = "AIzaSyBQxrh1ixO7oDDfGrBYxvpRkSoIl1RrNVk"
	googleMapsBase   = "https://maps.googleapis.com/maps/api/staticmap?"
)

// AttachmentBuffer holds a downloaded attachment PDF ready for merging.
type AttachmentBuffer struct {
	Content  []byte
	Filename string
	DocType  int
	SecIdx   int
}

// PDFResult is the return type of PackageCreateSearchPackage.
// Mirrors the JS return: searchPkgBuffer (preview) or { updatePdfBase64 } (final).
type PDFResult struct {
	IsPreview       bool
	PDFBuffer       []byte   // set when IsPreview == true
	UpdatePDFBase64 string   // base64 PDF when IsPreview == false
}

// ---------------------------------------------------------------------------
// generateHTMLTemplate — Go equivalent of pdf_helper.js exports.generateHTMLTemplate (L46-L197)
//
// Flow:
//  1. Populate template helpers on the payload (owners, dates, maps URLs, disclaimer)
//  2. Render search-package HTML → print to PDF via chromedp
//  3. Render cover-page HTML  → print to PDF via chromedp
//  4. Download attachment PDFs from S3
//  5. Merge: cover + report + attachments → final PDF via pdfcpu
// ---------------------------------------------------------------------------

func generateHTMLTemplate(
	ctx context.Context,
	payload *ReportPayload,
	isPreview bool,
	adminID string,
	sortedDocs []SortedDoc,
	s3c *S3Client,
) ([]byte, error) {
	// ── 1. Populate template helpers ─────────────────────────────────────────
	now := time.Now()
	payload.Owners = getCurrentOwnersList(payload.Report.SearchParties)
	payload.NamesSearched = getNamesSearchedList(payload.Report.SearchParties)
	payload.SearchDate = getSearchDate(now)
	payload.SearchTime = getSearchTime(now)
	payload.EstOrEdt = getEstOrEdt(now)

	if payload.Report.ReportDetails != nil {
		payload.EffectiveDate = formatDate(getStr(payload.Report.ReportDetails, "Effective_Date"))
		payload.EffectiveTime = formatTime12h(getStr(payload.Report.ReportDetails, "Effective_Date"))
		payload.YrsSearch = getStr(payload.Report.ReportDetails, "Years_Searched")
		payload.SearchStartDate = formatDate(getStr(payload.Report.ReportDetails, "Search_Start_Date"))
		payload.PriorEffectiveDate = formatDate(getStr(payload.Report.ReportDetails, "Prior_Effective_Date"))
		payload.IncludeDoesNotApply = getInt(payload.Report.ReportDetails, "Include_Does_Not_Apply")
	}

	if payload.OrderData != nil {
		payload.FooterTxt = payload.OrderData.ProductDescription.String
		if spv, _ := payload.Report.SPVersion.(string); spv != "" && spv != "1" {
			payload.FooterTxt += " - VERSION " + spv
		}
	}

	payload.GoogleMapImageSecond = buildMapsURL(payload, "450x260")
	payload.GoogleMapImage = buildMapsURL(payload, "520x243")
	payload.Disclaimer = readDisclaimerText(payload.OrderData)
	payload.AssetDir = assetsDir()

	// ── 2. Populate asset paths ──────────────────────────────────────────────
	absAssetDir, err := filepath.Abs(assetsDir())
	if err != nil {
		absAssetDir = assetsDir()
	}
	assetFile := func(name string) string {
		return "file://" + filepath.ToSlash(filepath.Join(absAssetDir, name))
	}

	payload.LogoPath = assetFile("Logo_Pippin.svg")
	payload.BuildingPath = assetFile("PippinBuildings.svg")
	payload.FooterPath = assetFile("footer2.svg")
	payload.FooterPathCust = assetFile("footer_cust.svg")
	payload.MenLogo = assetFile("images.jpeg")

	builderTemplate := getInt(payload.Report.ReportDetails, "Builder_Templ")
	if builderTemplate == 0 {
		builderTemplate = 1
	}

	if builderTemplate == 3 {
		payload.PippinReport = assetFile("pippin_report_1.svg")
		payload.SealPath = assetFile("Seal_1.svg")
		payload.EffectiveLogo = assetFile("Pippin_Effective_Seal_1.svg")
	} else {
		payload.PippinReport = assetFile("pippin_report.svg")
		payload.SealPath = assetFile("Seal.svg")
		payload.EffectiveLogo = assetFile("Pippin_Effective_Seal.svg")
	}

	// S3 custom logo if template is 2 or 3 and logo is set
	payload.CustLogo = assetFile("Logo_Pippin.svg")
	if (builderTemplate == 2 || builderTemplate == 3) && payload.OrderData != nil && payload.OrderData.Logo.Valid && payload.OrderData.Logo.String != "" {
		if s3c != nil {
			presigned, err := s3c.GetPresignedURL(ctx, payload.OrderData.Logo.String, 1*time.Hour)
			if err != nil {
				log.Printf("failed to get presigned URL for custom logo (non-fatal): %v", err)
			} else {
				payload.CustLogo = presigned
			}
		}
	}

	// Prepare EJS reportData object
	reportDataMap := OrderDataToMap(payload.OrderData)

	// ── 3. Render & print search-package report ───────────────────────────────
	reportHTML, err := renderTemplateEJS(filepath.Join(templateDir(), "searchPackagePdf.html"), payload, reportDataMap)
	if err != nil {
		return nil, fmt.Errorf("render report template: %w", err)
	}

	reportPDF, err := printPDF(ctx, reportHTML)
	if err != nil {
		return nil, fmt.Errorf("printPDF (report): %w", err)
	}

	// ── 4. Render & print cover page ──────────────────────────────────────────
	coverHTML, err := renderTemplateEJS(filepath.Join(templateDir(), "coverPageSave.html"), payload, reportDataMap)
	if err != nil {
		return nil, fmt.Errorf("render cover template: %w", err)
	}

	coverPDF, err := printPDF(ctx, coverHTML)
	if err != nil {
		return nil, fmt.Errorf("printPDF (cover): %w", err)
	}

	// ── 5. Download attachments from S3 ───────────────────────────────────────
	attachments, err := downloadAttachments(ctx, sortedDocs, s3c)
	if err != nil {
		// Non-fatal — proceed without attachments rather than failing the whole request
		log.Printf("downloadAttachments (non-fatal): %v", err)
		attachments = nil
	}

	log.Printf("coverPDF size: %d bytes, reportPDF size: %d bytes", len(coverPDF), len(reportPDF))
	// ── 6. Merge all PDFs ─────────────────────────────────────────────────────
	return mergePDFs(coverPDF, reportPDF, attachments)
}

// ---------------------------------------------------------------------------
// printPDF — Go equivalent of printPdf() in pdf_helper.js (L358-L390)
// Uses chromedp (headless Chrome) — same engine as Puppeteer.
// ---------------------------------------------------------------------------

func printPDF(ctx context.Context, html string) ([]byte, error) {
	// Write HTML to a temp file (same approach as the JS code)
	tmp, err := os.CreateTemp("", "rb-*.html")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmp.Name())

	log.Printf("printPDF: HTML length: %d", len(html))
	debugHTMLPath := "/Users/mokshchadha/rnd/go_rb/debug_print.html"
	if err := os.WriteFile(debugHTMLPath, []byte(html), 0644); err != nil {
		log.Printf("failed to write debug HTML: %v", err)
	} else {
		log.Printf("Saved debug HTML to %s", debugHTMLPath)
	}

	if _, err := tmp.WriteString(html); err != nil {
		tmp.Close()
		return nil, fmt.Errorf("write html: %w", err)
	}
	tmp.Close()

	// Allocate a headless Chrome instance
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx,
		chromedp.NoSandbox,
		chromedp.DisableGPU,
		chromedp.Headless,
		chromedp.Flag("disable-setuid-sandbox", true),
	)
	defer cancelAlloc()

	chromedpCtx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	var pdfBuf []byte

	// Mirror: page.pdf({ format:'letter', margin:{top:16px,right:20px,bottom:8px,left:20px},
	//                    preferCSSPageSize:true, printBackground:true })
	if err := chromedp.Run(chromedpCtx,
		chromedp.Navigate("file://"+tmp.Name()),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfBuf, _, err = page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.5).     // Letter width (inches)
				WithPaperHeight(11).     // Letter height (inches)
				WithMarginTop(0.167).    // ≈16px
				WithMarginRight(0.208).  // ≈20px
				WithMarginBottom(0.083). // ≈8px
				WithMarginLeft(0.208).   // ≈20px
				WithPreferCSSPageSize(true).
				Do(ctx)
			return err
		}),
	); err != nil {
		return nil, fmt.Errorf("chromedp run: %w", err)
	}

	return pdfBuf, nil
}

// ---------------------------------------------------------------------------
// mergePDFs — merges cover + report + attachments into one PDF.
// Uses gpdf; mirrors mergePDFDocuments() / mergeCPSP() in pdf_helper.js.
// ---------------------------------------------------------------------------

func mergePDFs(cover, report []byte, attachments []AttachmentBuffer) ([]byte, error) {
	sources := make([]gpdf.Source, 0, 2+len(attachments))
	sources = append(sources, gpdf.Source{Data: cover})
	sources = append(sources, gpdf.Source{Data: report})
	for _, att := range attachments {
		sources = append(sources, gpdf.Source{Data: att.Content})
	}

	merged, err := gpdf.Merge(sources)
	if err != nil {
		return nil, fmt.Errorf("gpdf Merge: %w", err)
	}
	return merged, nil
}

// ---------------------------------------------------------------------------
// downloadAttachments — mirrors getBufferFromS3Bucket() (pdf_helper.js L637-L672)
// ---------------------------------------------------------------------------

func downloadAttachments(ctx context.Context, docs []SortedDoc, s3c *S3Client) ([]AttachmentBuffer, error) {
	if s3c == nil || len(docs) == 0 {
		return nil, nil
	}
	var attachments []AttachmentBuffer
	for _, doc := range docs {
		s3Path := getStr(doc.Path, "Path")
		origName := getStr(doc.Path, "Original_Name")
		typeID := getInt(doc.Path, "Type_ID")

		content, err := s3c.DownloadFile(ctx, s3Path)
		if err != nil {
			log.Printf("s3 download %s: %v (skipped)", s3Path, err)
			continue
		}
		attachments = append(attachments, AttachmentBuffer{
			Content:  content,
			Filename: origName,
			DocType:  typeID,
			SecIdx:   doc.Idx,
		})
	}
	return attachments, nil
}

// ---------------------------------------------------------------------------
// Template rendering — uses Go html/template.
// NOTE: The EJS templates from the JS project must be converted to Go template
//       syntax. Place converted templates in the templates/master_template/ dir.
// ---------------------------------------------------------------------------

func renderTemplateEJS(tmplPath string, orders any, reportData any) (string, error) {
	inputMap := map[string]any{
		"templatePath": tmplPath,
		"orders":       orders,
		"reportData":   reportData,
	}
	inputBytes, err := json.Marshal(inputMap)
	if err != nil {
		return "", fmt.Errorf("marshal ejs input: %w", err)
	}

	renderScriptPath := "render.js"
	if stat, err := os.Stat("../render.js"); err == nil && !stat.IsDir() {
		renderScriptPath = "../render.js"
	} else if stat, err := os.Stat("render.js"); err != nil || stat.IsDir() {
		renderScriptPath = "render.js"
	}

	cmd := exec.Command("node", renderScriptPath)
	cmd.Stdin = bytes.NewReader(inputBytes)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("node render.js failed: %v, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

func OrderDataToMap(o *OrderPropertyData) map[string]any {
	if o == nil {
		return make(map[string]any)
	}
	return map[string]any{
		"File_ID":             o.FileID.String,
		"Order_ID":            o.OrderID.Int64,
		"completeAddress":     o.CompleteAddress,
		"Property_County":     o.PropertyCounty.String,
		"Property_State_Abbr": o.PropertyStateAbbr.String,
		"Property_City":       o.PropertyCity.String,
		"Product_Description": o.ProductDescription.String,
		"Report_Cover_Info":   o.ReportCoverInfo.String,
		"Organization_ID":     o.OrganizationID.String,
		"SP_DocInReport":      o.SPDocInReport.Int64,
		"Logo":                o.Logo.String,
	}
}

func templateDir() string {
	// 1. Check if templates/master_template exists in current working directory
	if stat, err := os.Stat(filepath.Join("templates", "master_template")); err == nil && stat.IsDir() {
		return filepath.Join("templates", "master_template")
	}
	// 2. Check if ../templates/master_template exists
	if stat, err := os.Stat(filepath.Join("..", "templates", "master_template")); err == nil && stat.IsDir() {
		return filepath.Join("..", "templates", "master_template")
	}
	// 3. Fallback to os.Executable()
	exe, err := os.Executable()
	if err != nil {
		return filepath.Join("templates", "master_template")
	}
	return filepath.Join(filepath.Dir(exe), "templates", "master_template")
}

func assetsDir() string {
	// 1. Check if assets exists in current working directory
	if stat, err := os.Stat("assets"); err == nil && stat.IsDir() {
		return "assets"
	}
	// 2. Check if ../assets exists
	if stat, err := os.Stat(filepath.Join("..", "assets")); err == nil && stat.IsDir() {
		return filepath.Join("..", "assets")
	}
	// 3. Fallback to os.Executable()
	exe, err := os.Executable()
	if err != nil {
		return "assets"
	}
	return filepath.Join(filepath.Dir(exe), "assets")
}

// ---------------------------------------------------------------------------
// Template data helpers
// ---------------------------------------------------------------------------

// getCurrentOwnersList mirrors getCurrentOwnersList() (pdf_helper.js L727-L753)
func getCurrentOwnersList(parties []map[string]any) []string {
	owners := []string{}
	for _, p := range parties {
		if getInt(p, "Is_Current_Owner") == 0 {
			continue
		}
		entityID := getInt(p, "Entity_ID")
		var name string
		switch entityID {
		case 1:
			parts := []string{
				strings.TrimSpace(getStr(p, "First_Business_Name")),
				strings.TrimSpace(getStr(p, "Middle_Name")),
				strings.TrimSpace(getStr(p, "Last_Name")),
			}
			var nonEmpty []string
			for _, pt := range parts {
				if pt != "" {
					nonEmpty = append(nonEmpty, pt)
				}
			}
			name = strings.Join(nonEmpty, " ")
		case 3:
			name = strings.TrimSpace(getStr(p, "First_Business_Name"))
		}
		if name != "" {
			owners = append(owners, "     "+name)
		}
	}
	return owners
}

// getNamesSearchedList mirrors getNamesSearchedList() (pdf_helper.js L755-L813)
func getNamesSearchedList(parties []map[string]any) []NamesSearchedItem {
	list := []NamesSearchedItem{}
	for _, p := range parties {
		aliases, _ := p["_aliases"].([]map[string]any)
		for _, alias := range aliases {
			if getInt(alias, "Is_Judgments_Liens") == 0 {
				continue
			}
			judgFound := getInt(alias, "Judgements_Found")
			lienFound := getInt(alias, "Liens_Found")
			var judgLabel, lienLabel string
			switch {
			case judgFound == 0 && lienFound == 0:
				judgLabel, lienLabel = "No Judgments", "Liens"
			case judgFound > 0 && lienFound > 0:
				jSuffix := "Judgment"
				if judgFound > 1 {
					jSuffix = "Judgments"
				}
				lSuffix := "Lien"
				if lienFound > 1 {
					lSuffix = "Liens"
				}
				judgLabel = fmt.Sprintf("%d %s", judgFound, jSuffix)
				lienLabel = fmt.Sprintf(" %d %s", lienFound, lSuffix)
			default:
				jSuffix := "Judgment"
				if judgFound > 1 {
					jSuffix = "Judgments"
				}
				if judgFound > 0 {
					judgLabel = fmt.Sprintf("%d %s", judgFound, jSuffix)
				} else {
					judgLabel = "No Judgments"
				}
				lSuffix := "Lien"
				if lienFound > 1 {
					lSuffix = "Liens"
				}
				if lienFound > 0 {
					lienLabel = fmt.Sprintf(" %d %s", lienFound, lSuffix)
				} else {
					lienLabel = "No Liens"
				}
			}

			isLegal := getInt(alias, "IsLegal")
			var name string
			if isLegal == 1 {
				name = constructLegalName(p)
			} else {
				name = strings.TrimSpace(getStr(alias, "Alias"))
			}
			if name != "" {
				list = append(list, NamesSearchedItem{
					Name:      name,
					JudgLabel: judgLabel,
					LienLabel: lienLabel,
					Judgments: judgFound,
					Liens:     lienFound,
				})
			}
		}
	}
	return list
}

// constructLegalName mirrors constructLegalName() (pdf_helper.js L815-L834)
func constructLegalName(p map[string]any) string {
	entityID := getInt(p, "Entity_ID")
	var name string
	switch entityID {
	case 1, 2:
		parts := []string{
			strings.TrimSpace(getStr(p, "First_Business_Name")),
			strings.TrimSpace(getStr(p, "Middle_Name")),
			strings.TrimSpace(getStr(p, "Last_Name")),
		}
		var nonEmpty []string
		for _, pt := range parts {
			if pt != "" {
				nonEmpty = append(nonEmpty, pt)
			}
		}
		name = strings.Join(nonEmpty, " ")
	case 3, 4:
		name = strings.TrimSpace(getStr(p, "First_Business_Name"))
	}
	return name
}

// ---------------------------------------------------------------------------
// Google Maps URL — mirrors the googleImage construction in pdf_helper.js
// ---------------------------------------------------------------------------

func buildMapsURL(payload *ReportPayload, size string) string {
	if payload.Report.ReportDetails != nil {
		lat := getStr(payload.Report.ReportDetails, "Report_Lat")
		lng := getStr(payload.Report.ReportDetails, "Report_Lng")
		if lat != "" && lng != "" {
			return fmt.Sprintf(
				"%scenter=%s,%s&markers=%s,%s&style=feature:poi|visibility:off&zoom=13&size=%s&scale=2&sensor=false&key=%s",
				googleMapsBase, lat, lng, lat, lng, size, googleMapsAPIKey,
			)
		}
	}
	if payload.OrderData != nil {
		city := payload.OrderData.PropertyAddress1.String + "+" +
			payload.OrderData.PropertyCounty.String + "+" +
			payload.OrderData.PropertyStateAbbr.String
		return fmt.Sprintf(
			"%scenter=%s&markers=|%s&style=feature:poi|visibility:off&zoom=13&size=%s&scale=2&sensor=false&key=%s",
			googleMapsBase, city, city, size, googleMapsAPIKey,
		)
	}
	return ""
}

// readDisclaimerText mirrors the disclaimer loading in pdf_helper.js (L159-L167).
func readDisclaimerText(orderData *OrderPropertyData) string {
	dir := assetsDir()
	fileName := "disclaimer.txt"
	if orderData != nil {
		if orgID := orderData.OrganizationID.String; orgID != "" && orgID == os.Getenv("TITLE_WRITE_ORG") {
			fileName = "title_write_disclaimer.txt"
		}
	}
	content, err := os.ReadFile(filepath.Join(dir, fileName))
	if err != nil {
		log.Printf("readDisclaimerText %s: %v", fileName, err)
		return ""
	}
	return linkifyURLs(string(content))
}
