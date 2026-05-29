// Package rb — search-package coordinator.
// Mirrors PackageHelper.createSearchPackage() from package_helper.js (L7-L52).
package rb

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
)

// PackageCreateSearchPackage is the Go equivalent of PackageHelper.createSearchPackage
// in package_helper.js (L7-L52).
//
// Flow (mirrors the JS exactly):
//  1. optimizeReportData — filter/cleanse all report sections concurrently
//  2. Attach version data + filter empty report changes
//  3. Fetch Organisation Report_Cover_Info (strip if empty HTML)
//  4. sortDocuments — build the ordered attachment list
//  5. generateHTMLTemplate — render EJS→HTML→PDF via chromedp, merge with S3 attachments
//  6. Return base64 PDF (final) or raw buffer (preview)
func (a *App) PackageCreateSearchPackage(
	ctx context.Context,
	payload *SearchPackagePayload,
	req CreateSearchPackageRequest,
) (*PDFResult, error) {

	// ── 1. Optimize / cleanse report data ────────────────────────────────────
	optimized, err := optimizeReportData(payload)
	if err != nil {
		return nil, fmt.Errorf("optimizeReportData: %w", err)
	}

	// ── 2. Apply version data & filter empty report changes ───────────────────
	// Mirrors: optimizedData.reportChanges = payload.reportChanges.filter(item => item != '')
	//          optimizedData.SP_Version = payload.versionData.SP_Version
	optimized.ReportChanges = filterEmptyStrings(optimized.ReportChanges)
	if payload.VersionData.SPVersion.Valid {
		optimized.SPVersion = payload.VersionData.SPVersion.String
	} else {
		optimized.SPVersion = ""
	}

	// ── 3. Org Report_Cover_Info ──────────────────────────────────────────────
	// Mirrors: if (payload?.orderData?.Organization_ID) { ... }
	if payload.OrderData.OrganizationID.Valid && payload.OrderData.OrganizationID.String != "" {
		coverInfo, err := a.getOrgReportCoverInfo(ctx, payload.OrderData.OrganizationID.String)
		if err != nil {
			log.Printf("getOrgReportCoverInfo (non-fatal): %v", err)
		} else {
			payload.OrderData.ReportCoverInfo = toNullString(coverInfo)
		}
	}

	// ── 4. Sort document attachment list ─────────────────────────────────────
	sortedDocs := sortDocuments(
		payload.BuilderData.SPDocuments,
		payload.PrepData.DocTypes,
		optimized,
	)

	// ── 5. Build ReportPayload for template ───────────────────────────────────
	reportPayload := &ReportPayload{
		Report:      *optimized,
		OrderData:   payload.OrderData,
		BuilderData: payload.BuilderData,
		PrepData:    payload.PrepData,
		VersionData: payload.VersionData,
	}

	// ── 6. Generate PDF ───────────────────────────────────────────────────────
	// Mirrors: PdfHelper.generateHTMLTemplate(curatedData, payload.orderData,
	//              req.body.isPreview, req.body.Admin_User_ID, sortedDocList)
	pdfBytes, err := generateHTMLTemplate(
		ctx,
		reportPayload,
		req.IsPreview,
		req.AdminUserID,
		sortedDocs,
		a.S3,
	)
	if err != nil {
		return nil, fmt.Errorf("generateHTMLTemplate: %w", err)
	}

	// Mirrors: if (req.body.isPreview) return searchPkgBuffer
	//          return { updatePdfBase64: searchPkgBuffer.toString('base64') }
	if req.IsPreview {
		return &PDFResult{IsPreview: true, PDFBuffer: pdfBytes}, nil
	}
	return &PDFResult{
		IsPreview:       false,
		UpdatePDFBase64: base64.StdEncoding.EncodeToString(pdfBytes),
	}, nil
}

// getOrgReportCoverInfo fetches Report_Cover_Info for an organisation and
// returns empty string if the HTML contains only whitespace/tags.
// Mirrors the org fetch block in package_helper.js (L22-L33).
func (a *App) getOrgReportCoverInfo(ctx context.Context, orgID string) (string, error) {
	var rawHTML sql.NullString
	err := a.DB.QueryRowContext(ctx,
		`SELECT Report_Cover_Info FROM Organizations WHERE Organization_ID = ? LIMIT 1`,
		orgID,
	).Scan(&rawHTML)
	if err != nil {
		return "", fmt.Errorf("query Organizations: %w", err)
	}
	raw := rawHTML.String
	if len(cleanHTMLForCheck(raw)) == 0 {
		return "", nil // treat as empty — matches JS `cleanedText.length > 0` check
	}
	return raw, nil
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
