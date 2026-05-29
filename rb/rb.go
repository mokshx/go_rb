// Package rb implements the report-builder search-package creation flow,
// translating the original Node.js/Sequelize logic into idiomatic Go.
//
// Key mapping from JS → Go:
//
//	Promise.all([...])         → golang.org/x/sync/errgroup (concurrent)
//	sequelize.transaction()    → *sql.Tx with defer rollback pattern
//	req.params / req.body      → CreateSearchPackageRequest struct
package rb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

// App holds the shared database connection and S3 client used across all handlers.
type App struct {
	DB *sql.DB
	S3 *S3Client // optional; nil disables attachment embedding
}

// ---------------------------------------------------------------------------
// Public entry point
// ---------------------------------------------------------------------------

// CreateSearchPackage is the Go equivalent of the JS exports.createSearchPackage
// function (helper.js L1604-L1634) + PackageHelper.createSearchPackage (package_helper.js L7-L52).
//
// Full end-to-end flow:
//  1. Concurrently fetch order data, builder data, prep/lookup data, and version
//     (mirrors Promise.all).
//  2. Build the complete address string.
//  3. Persist the builder JSON snapshot inside a DB transaction (saveJSONData).
//  4. Optimise/cleanse all report sections (optimizeReportData).
//  5. Render HTML templates and print PDFs via chromedp (headless Chrome).
//  6. Merge PDFs (cover + report + S3 attachments) via pdfcpu.
//  7. Return base64 PDF (non-preview) or raw buffer (preview).
func (a *App) CreateSearchPackage(ctx context.Context, req CreateSearchPackageRequest) (*PDFResult, error) {
	// ── Step 1: Concurrent data fetch ────────────────────────────────────────
	var (
		orderData   OrderPropertyData
		builderData BuilderData
		prepData    PrepData
		versionData ReportVersion
	)

	g := new(errgroup.Group)

	g.Go(func() error {
		data, err := a.getOrderPropertyByID(req.OrderID)
		if err != nil {
			return fmt.Errorf("getOrderPropertyByID: %w", err)
		}
		orderData = data
		return nil
	})

	g.Go(func() error {
		data, err := a.getBuilderData(req.BuilderID)
		if err != nil {
			return fmt.Errorf("getBuilderData: %w", err)
		}
		builderData = data
		return nil
	})

	g.Go(func() error {
		data, err := a.collectPrepData()
		if err != nil {
			return fmt.Errorf("collectPrepData: %w", err)
		}
		prepData = data
		return nil
	})

	g.Go(func() error {
		data, err := a.getVersionDetailsByBuilderID(req.BuilderID)
		if err != nil {
			return fmt.Errorf("getVersionDetailsByBuilderID: %w", err)
		}
		versionData = data
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// ── Step 2: Enrich order data with a formatted complete address ───────────
	orderData.CompleteAddress = buildAddress(orderData)

	// ── Step 3: Persist the builder JSON snapshot in a transaction ───────────
	if err := a.saveJSONData(builderData, req.AdminUserID); err != nil {
		return nil, fmt.Errorf("saveJSONData: %w", err)
	}

	// ── Step 4: Assemble payload ──────────────────────────────────────────────
	payload := &SearchPackagePayload{
		OrderData:   &orderData,
		BuilderData: &builderData,
		VersionData: &versionData,
		PrepData:    &prepData,
	}

	// ── Step 5: Run the full PDF generation pipeline ─────────────────────────
	// Mirrors: return PackageHelper.createSearchPackage(payload, req)
	return a.PackageCreateSearchPackage(ctx, payload, req)
}

// ---------------------------------------------------------------------------
// buildAddress mirrors the JS buildAddress() helper (helper.js L1992-L2028)
//
// Rules:
//   - City + state only   → "City, ST [ZIP]"
//   - Full address        → "Addr1[\nAddr2]\nCity, ST ZIP"
//   - Anything else       → ""
//
// ---------------------------------------------------------------------------
func buildAddress(o OrderPropertyData) string {
	addr1 := o.PropertyAddress1.String
	addr2 := o.PropertyAddress2.String
	city := o.PropertyCity.String
	state := o.PropertyStateAbbr.String
	zip := o.PropertyZipCode.String

	// Case 1: no street address but city + state are present
	if addr1 == "" && city != "" && state != "" {
		result := city + ", " + state
		if zip != "" {
			result += " " + zip
		}
		return result
	}

	// Case 2: full address (street + city at minimum)
	if addr1 != "" && city != "" {
		var parts []string
		parts = append(parts, addr1)
		if addr2 != "" {
			parts = append(parts, addr2)
		}
		cityState := city
		if state != "" {
			cityState += ", " + state
		}
		if zip != "" {
			cityState += " " + zip
		}
		parts = append(parts, cityState)
		return strings.Join(parts, "\n")
	}

	return ""
}

// ---------------------------------------------------------------------------
// getBuilderData mirrors the JS getBuilderData() function (helper.js L1766-L1805)
//
// All sections are fetched concurrently; parties include their aliases.
// ---------------------------------------------------------------------------
func (a *App) getBuilderData(builderID string) (BuilderData, error) {
	var (
		searchPackage               SearchPackage
		parties                     []SPParty
		assessments                 []map[string]any
		taxes                       []map[string]any
		chainVD                     []map[string]any
		chainCT                     []map[string]any
		securityInstruments         []map[string]any
		lienJudgements              []map[string]any
		exceptionRestrictionAdverse []map[string]any
		globalCommitments           []map[string]any
		commitmentTypings           []map[string]any
		documents                   []map[string]any
		generalComments             []map[string]any
	)

	g := new(errgroup.Group)

	g.Go(func() error {
		data, err := a.getSearchPackageByBuilderID(builderID)
		if err != nil {
			return fmt.Errorf("getSearchPackageByBuilderID: %w", err)
		}
		searchPackage = data
		return nil
	})

	g.Go(func() error {
		data, err := a.getSearchPartiesWithAliases(builderID)
		if err != nil {
			return fmt.Errorf("getSearchPartiesWithAliases: %w", err)
		}
		parties = data
		return nil
	})

	g.Go(func() error {
		data, err := a.queryGenericRows("SELECT * FROM SP_Assessments WHERE Sp_Id = ?", builderID)
		if err != nil {
			return fmt.Errorf("SP_Assessments: %w", err)
		}
		assessments = data
		return nil
	})

	g.Go(func() error {
		data, err := a.queryGenericRows("SELECT * FROM SP_Taxes WHERE Sp_Id = ?", builderID)
		if err != nil {
			return fmt.Errorf("SP_Taxes: %w", err)
		}
		taxes = data
		return nil
	})

	g.Go(func() error {
		data, err := a.queryGenericRows(
			"SELECT * FROM SP_Chain_Of_Titles WHERE Sp_Id = ? AND Is_Vesting_Deed = 1", builderID)
		if err != nil {
			return fmt.Errorf("SP_Chain_Of_Titles VD: %w", err)
		}
		chainVD = data
		return nil
	})

	g.Go(func() error {
		data, err := a.queryGenericRows(
			"SELECT * FROM SP_Chain_Of_Titles WHERE Sp_Id = ? AND Is_Vesting_Deed = 0", builderID)
		if err != nil {
			return fmt.Errorf("SP_Chain_Of_Titles CT: %w", err)
		}
		chainCT = data
		return nil
	})

	g.Go(func() error {
		data, err := a.queryGenericRows("SELECT * FROM SP_Security_Instruments WHERE Sp_Id = ?", builderID)
		if err != nil {
			return fmt.Errorf("SP_Security_Instruments: %w", err)
		}
		securityInstruments = data
		return nil
	})

	g.Go(func() error {
		data, err := a.queryGenericRows("SELECT * FROM SP_Lien_Judgement WHERE Sp_Id = ?", builderID)
		if err != nil {
			return fmt.Errorf("SP_Lien_Judgement: %w", err)
		}
		lienJudgements = data
		return nil
	})

	g.Go(func() error {
		data, err := a.queryGenericRows(
			"SELECT * FROM SP_Exception_Restriction_Adverse WHERE Sp_Id = ?", builderID)
		if err != nil {
			return fmt.Errorf("SP_Exception_Restriction_Adverse: %w", err)
		}
		exceptionRestrictionAdverse = data
		return nil
	})

	g.Go(func() error {
		data, err := a.queryGenericRows("SELECT * FROM Global_Commitments WHERE Sp_Id = ?", builderID)
		if err != nil {
			return fmt.Errorf("Global_Commitments: %w", err)
		}
		globalCommitments = data
		return nil
	})

	g.Go(func() error {
		data, err := a.queryGenericRows("SELECT * FROM Commitment_Typings WHERE Sp_Id = ?", builderID)
		if err != nil {
			return fmt.Errorf("Commitment_Typings: %w", err)
		}
		commitmentTypings = data
		return nil
	})

	g.Go(func() error {
		data, err := a.queryGenericRows("SELECT * FROM SP_Documents WHERE Sp_Id = ?", builderID)
		if err != nil {
			return fmt.Errorf("SP_Documents: %w", err)
		}
		documents = data
		return nil
	})

	g.Go(func() error {
		data, err := a.queryGenericRows("SELECT * FROM SP_General_Comments WHERE Sp_Id = ?", builderID)
		if err != nil {
			return fmt.Errorf("SP_General_Comments: %w", err)
		}
		generalComments = data
		return nil
	})

	if err := g.Wait(); err != nil {
		return BuilderData{}, err
	}

	return BuilderData{
		SearchPackage:                 searchPackage,
		SPParties:                     parties,
		SPAssessments:                 assessments,
		SPTaxes:                       taxes,
		SPChainOfTitlesVD:             chainVD,
		SPChainOfTitlesCT:             chainCT,
		SPSecurityInstruments:         securityInstruments,
		SPLienJudgement:               lienJudgements,
		SPExceptionRestrictionAdverse: exceptionRestrictionAdverse,
		GlobalCommitments:             globalCommitments,
		CommitmentTypings:             commitmentTypings,
		SPDocuments:                   documents,
		SPGeneralComments:             generalComments,
	}, nil
}

// getSearchPartiesWithAliases mirrors getSearchParties() (helper.js L1807-L1818).
// It fetches all active parties then, concurrently, fetches each party's aliases.
func (a *App) getSearchPartiesWithAliases(builderID string) ([]SPParty, error) {
	parties, err := a.getActiveParties(builderID)
	if err != nil {
		return nil, err
	}
	if len(parties) == 0 {
		return parties, nil
	}

	// Fetch aliases for all parties concurrently.
	aliases := make([][]SPPartyAlias, len(parties))
	g := new(errgroup.Group)

	for i, party := range parties {
		i, party := i, party // capture loop vars
		g.Go(func() error {
			a, err := a.getActiveAliasesForParty(party.ID)
			if err != nil {
				return fmt.Errorf("getActiveAliasesForParty(partyID=%d): %w", party.ID, err)
			}
			aliases[i] = a
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	for i := range parties {
		parties[i].Alias = aliases[i]
	}
	return parties, nil
}

// ---------------------------------------------------------------------------
// collectPrepData mirrors the JS collectPrepData() function (helper.js L1662-L1701)
//
// All 13 lookup tables are queried concurrently and assembled into PrepData.
// ---------------------------------------------------------------------------
func (a *App) collectPrepData() (PrepData, error) {
	var (
		docTypes         []DocumentType
		taxSources       []TaxType
		taxStatus        []TaxPaidStatus
		chainTypes       []ChainInstrumentType
		lienTypes        []LienJudgementType
		partyEntities    []PartyEntities
		secInstEntities  []SecurityInstrumentsType
		eraEntities      []ERAEntities
		taxEntities      []TaxEntities
		chainEntities    []ChainOfTitleEntities
		lienEntities     []LienJudgementEntities
		taxAuthorityType []TaxAuthorityType
		interestTypes    []PropertyInterestType
	)

	g := new(errgroup.Group)

	g.Go(func() error {
		m := a.getDocTypeIdMap()
		for _, v := range m {
			docTypes = append(docTypes, v)
		}
		return nil
	})

	g.Go(func() error {
		m := a.getTaxSourceTypeIdMap()
		for _, v := range m {
			taxSources = append(taxSources, v)
		}
		return nil
	})

	g.Go(func() error {
		m := a.getTaxPaidStatusMap()
		for _, v := range m {
			taxStatus = append(taxStatus, v)
		}
		return nil
	})

	g.Go(func() error {
		m := a.getChainTypesIdMap()
		for _, v := range m {
			chainTypes = append(chainTypes, v)
		}
		return nil
	})

	g.Go(func() error {
		m := a.getLienTypesIdMap()
		for _, v := range m {
			lienTypes = append(lienTypes, v)
		}
		return nil
	})

	g.Go(func() error {
		m := a.getSPPartyIdMap()
		for _, v := range m {
			partyEntities = append(partyEntities, v)
		}
		return nil
	})

	g.Go(func() error {
		m := a.getSecInstrumentsIdMap()
		for _, v := range m {
			secInstEntities = append(secInstEntities, v)
		}
		return nil
	})

	g.Go(func() error {
		m := a.getEraIdMap()
		for _, v := range m {
			eraEntities = append(eraEntities, v)
		}
		return nil
	})

	g.Go(func() error {
		m := a.getTaxEntitiesIdMap()
		for _, v := range m {
			taxEntities = append(taxEntities, v)
		}
		return nil
	})

	g.Go(func() error {
		m := a.getChainOfTitleTypeIdMap()
		for _, v := range m {
			chainEntities = append(chainEntities, v)
		}
		return nil
	})

	g.Go(func() error {
		m := a.getLienJudgementIdMap()
		for _, v := range m {
			lienEntities = append(lienEntities, v)
		}
		return nil
	})

	g.Go(func() error {
		m := a.getTaxAuthorityTypeMap()
		for _, v := range m {
			taxAuthorityType = append(taxAuthorityType, v)
		}
		return nil
	})

	g.Go(func() error {
		m := a.getPropertyInterestTypes()
		for _, v := range m {
			interestTypes = append(interestTypes, v)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return PrepData{}, err
	}

	return PrepData{
		DocTypes:         docTypes,
		TaxSources:       taxSources,
		TaxStatus:        taxStatus,
		ChainTypes:       chainTypes,
		LienTypes:        lienTypes,
		PartyEntities:    partyEntities,
		SecInstEntities:  secInstEntities,
		ERAEntities:      eraEntities,
		TaxEntities:      taxEntities,
		ChainEntities:    chainEntities,
		LienEntities:     lienEntities,
		TaxAuthorityType: taxAuthorityType,
		InterestTypes:    interestTypes,
	}, nil
}

// ---------------------------------------------------------------------------
// saveJSONData mirrors the JS saveJSONData() function (helper.js L1820-L1827)
//
// It serialises the current builder snapshot to JSON and persists it inside
// a database transaction so the version record stays consistent.
// ---------------------------------------------------------------------------
func (a *App) saveJSONData(builder BuilderData, userID string) error {
	jsonBytes, err := json.Marshal(builder)
	if err != nil {
		return fmt.Errorf("marshal builder data: %w", err)
	}

	// Fetch the current version record for the search package.
	version, err := a.getVersionDetailsByBuilderID(fmt.Sprintf("%d", builder.SearchPackage.ID))
	if err != nil {
		return fmt.Errorf("getVersionDetailsByBuilderID: %w", err)
	}

	// Persist inside a transaction — mirrors sequelize.transaction({ autocommit:false }).
	tx, err := a.DB.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	now := time.Now().UTC().Format("2006-01-02 15:04:05")

	_, err = tx.Exec(
		`UPDATE Report_Versions
		    SET Report_JSON     = ?,
		        Json_Update_At  = ?,
		        Modified_At     = ?,
		        Modified_By     = ?
		  WHERE Id = ?`,
		string(jsonBytes),
		now,
		now,
		userID,
		version.ID,
	)
	if err != nil {
		return fmt.Errorf("update Report_Versions: %w", err)
	}

	return tx.Commit()
}

// ---------------------------------------------------------------------------
// Repository helpers — thin wrappers around raw SQL
// ---------------------------------------------------------------------------

// getOrderPropertyByID fetches a single row from Order_Property_User_View.
func (a *App) getOrderPropertyByID(orderID string) (OrderPropertyData, error) {
	const query = `
		SELECT
			Property_ID,
			Order_ID,
			Customer_ID,
			File_ID,
			Property_Address_1,
			Property_Address_2,
			Property_City,
			Property_County,
			Property_State,
			Property_State_Abbr,
			Property_ZipCode,
			Property_Latitude,
			Property_Longitude,
			Order_Creation_Date,
			Order_Modification_Date,
			Order_Status,
			Order_Product_Status,
			Product_ID,
			Product_Description,
			Organization_ID,
			Organization_Name,
			User_First_Name,
			User_Last_Name,
			User_Full_Name,
			User_Login_Name,
			SP_DocInReport
		FROM Order_Property_User_View
		WHERE Order_ID = ?
	`

	var d OrderPropertyData
	err := a.DB.QueryRow(query, orderID).Scan(
		&d.PropertyID,
		&d.OrderID,
		&d.CustomerID,
		&d.FileID,
		&d.PropertyAddress1,
		&d.PropertyAddress2,
		&d.PropertyCity,
		&d.PropertyCounty,
		&d.PropertyState,
		&d.PropertyStateAbbr,
		&d.PropertyZipCode,
		&d.PropertyLatitude,
		&d.PropertyLongitude,
		&d.OrderCreationDate,
		&d.OrderModificationDate,
		&d.OrderStatus,
		&d.OrderProductStatus,
		&d.ProductID,
		&d.ProductDescription,
		&d.OrganizationID,
		&d.OrganizationName,
		&d.UserFirstName,
		&d.UserLastName,
		&d.UserFullName,
		&d.UserLoginName,
		&d.SPDocInReport,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return OrderPropertyData{}, nil
		}
		return OrderPropertyData{}, fmt.Errorf("query Order_Property_User_View: %w", err)
	}
	return d, nil
}

// getSearchPackageByBuilderID fetches the Report_Builder row for a builder.
func (a *App) getSearchPackageByBuilderID(builderID string) (SearchPackage, error) {
	const query = `
		SELECT
			Id, Order_ID, VD_Manual_Sort, CT_Manual_Sort,
			SI_Manual_Sort, LJ_Manual_Sort, ER_Manual_Sort,
			PUD, Parcel, Registry, Interest_Type_Id,
			Include_Does_Not_Apply
		FROM Report_Builder
		WHERE Id = ?
	`
	var sp SearchPackage
	err := a.DB.QueryRow(query, builderID).Scan(
		&sp.ID,
		&sp.OrderID,
		&sp.VDManualSort,
		&sp.CTManualSort,
		&sp.SIManualSort,
		&sp.LJManualSort,
		&sp.ERManualSort,
		&sp.PUD,
		&sp.Parcel,
		&sp.Registry,
		&sp.InterestTypeID,
		&sp.IncludeDoesNotApply,
	)
	if err != nil && err != sql.ErrNoRows {
		return SearchPackage{}, fmt.Errorf("query Report_Builder: %w", err)
	}
	return sp, nil
}

// getVersionDetailsByBuilderID mirrors Report_Versions.getVersionDetailsByBuilderId().
func (a *App) getVersionDetailsByBuilderID(builderID string) (ReportVersion, error) {
	const query = `
		SELECT Id, Sp_Id, SP_Version, Report_JSON, Json_Update_At,
		       Modified_At, Created_At, Created_By, Modified_By
		  FROM Report_Versions
		 WHERE Sp_Id = ?
		 ORDER BY Id DESC
		 LIMIT 1
	`
	var rv ReportVersion
	err := a.DB.QueryRow(query, builderID).Scan(
		&rv.ID,
		&rv.SpID,
		&rv.SPVersion,
		&rv.JsonData,
		&rv.JsonUpdatAt,
		&rv.ModifiedAt,
		&rv.CreatedAt,
		&rv.CreatedBy,
		&rv.ModifiedBy,
	)
	if err != nil && err != sql.ErrNoRows {
		return ReportVersion{}, fmt.Errorf("query Report_Versions: %w", err)
	}
	return rv, nil
}

// getActiveParties returns all Status=1 parties for a builder.
func (a *App) getActiveParties(builderID string) ([]SPParty, error) {
	rows, err := a.DB.Query(
		`SELECT Id, Sp_Id, Entity_ID, First_Business_Name, Middle_Name, Last_Name, Status, Applies
		   FROM SP_Parties
		  WHERE Sp_Id = ? AND Status = 1`, builderID)
	if err != nil {
		return nil, fmt.Errorf("query SP_Parties: %w", err)
	}
	defer rows.Close()

	var parties []SPParty
	for rows.Next() {
		var p SPParty
		if err := rows.Scan(
			&p.ID, &p.SpID, &p.EntityID,
			&p.FirstBusinessName, &p.MiddleName, &p.LastName,
			&p.Status, &p.Applies,
		); err != nil {
			return nil, fmt.Errorf("scan SP_Parties: %w", err)
		}
		parties = append(parties, p)
	}
	return parties, rows.Err()
}

// getActiveAliasesForParty returns all Status=1 aliases for a party.
func (a *App) getActiveAliasesForParty(partyID int) ([]SPPartyAlias, error) {
	rows, err := a.DB.Query(
		`SELECT Id, Party_ID, Alias, IsLegal, Status
		   FROM SP_Parties_Alias
		  WHERE Party_ID = ? AND Status = 1`, partyID)
	if err != nil {
		return nil, fmt.Errorf("query SP_Parties_Alias: %w", err)
	}
	defer rows.Close()

	var aliases []SPPartyAlias
	for rows.Next() {
		var al SPPartyAlias
		if err := rows.Scan(&al.ID, &al.PartyID, &al.Alias, &al.IsLegal, &al.Status); err != nil {
			return nil, fmt.Errorf("scan SP_Parties_Alias: %w", err)
		}
		aliases = append(aliases, al)
	}
	return aliases, rows.Err()
}

// queryGenericRows executes a query and returns each row as map[string]any,
// keeping values as strings (matching the JS behaviour where Sequelize models
// return plain objects). Used for sections where we don't need typed structs.
func (a *App) queryGenericRows(query string, args ...any) ([]map[string]any, error) {
	rows, err := a.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("columns: %w", err)
	}

	var result []map[string]any
	for rows.Next() {
		values := make([]any, len(cols))
		valuePtrs := make([]any, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}

		row := make(map[string]any, len(cols))
		for i, col := range cols {
			if b, ok := values[i].([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = values[i]
			}
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// ---------------------------------------------------------------------------
// Lookup-table map builders (used by collectPrepData and getAllMaps)
// ---------------------------------------------------------------------------

func (a *App) getDocTypeIdMap() map[int]DocumentType {
	m := make(map[int]DocumentType)
	rows, err := a.DB.Query("SELECT Type, Id, Type_Status FROM SP_Document_Types")
	if err != nil {
		log.Printf("getDocTypeIdMap: %v", err)
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var d DocumentType
		if err := rows.Scan(&d.Type, &d.Id, &d.TypeStatus); err == nil && d.TypeStatus {
			m[d.Id] = d
		}
	}
	return m
}

func (a *App) getTaxSourceTypeIdMap() map[int]TaxType {
	m := make(map[int]TaxType)
	rows, err := a.DB.Query("SELECT Id, Source_Type, Source_Type_Status FROM SP_Tax_Source_Types")
	if err != nil {
		log.Printf("getTaxSourceTypeIdMap: %v", err)
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var d TaxType
		if err := rows.Scan(&d.Id, &d.SourceType, &d.TypeStatus); err == nil && d.TypeStatus {
			m[d.Id] = d
		}
	}
	return m
}

func (a *App) getTaxPaidStatusMap() map[int]TaxPaidStatus {
	m := make(map[int]TaxPaidStatus)
	rows, err := a.DB.Query("SELECT Id, Status, Status_Type FROM SP_Tax_Paid_Status_Types")
	if err != nil {
		log.Printf("getTaxPaidStatusMap: %v", err)
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var t TaxPaidStatus
		if err := rows.Scan(&t.Id, &t.Status, &t.StatusType); err == nil && t.Status {
			m[t.Id] = t
		}
	}
	return m
}

func (a *App) getChainTypesIdMap() map[int]ChainInstrumentType {
	m := make(map[int]ChainInstrumentType)
	rows, err := a.DB.Query("SELECT Id, Type, Type_ID, Status FROM SP_Chain_Of_Title_Instrument_Types")
	if err != nil {
		log.Printf("getChainTypesIdMap: %v", err)
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var d ChainInstrumentType
		if err := rows.Scan(&d.Id, &d.Type, &d.TypeId, &d.Status); err == nil && d.Status {
			m[d.Id] = d
		}
	}
	return m
}

func (a *App) getLienTypesIdMap() map[int]LienJudgementType {
	m := make(map[int]LienJudgementType)
	rows, err := a.DB.Query("SELECT Id, Type, Type_ID, Status FROM SP_Lien_Judgement_Types")
	if err != nil {
		log.Printf("getLienTypesIdMap: %v", err)
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var lj LienJudgementType
		if err := rows.Scan(&lj.Id, &lj.Type, &lj.TypeId, &lj.Status); err == nil && lj.Status {
			m[lj.Id] = lj
		}
	}
	return m
}

func (a *App) getSPPartyIdMap() map[int]PartyEntities {
	m := make(map[int]PartyEntities)
	rows, err := a.DB.Query("SELECT Id, Entity, Entity_Status FROM SP_Party_Entities")
	if err != nil {
		log.Printf("getSPPartyIdMap: %v", err)
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var p PartyEntities
		if err := rows.Scan(&p.Id, &p.Entity, &p.EntityStatus); err == nil && p.EntityStatus {
			m[p.Id] = p
		}
	}
	return m
}

func (a *App) getSecInstrumentsIdMap() map[int]SecurityInstrumentsType {
	m := make(map[int]SecurityInstrumentsType)
	rows, err := a.DB.Query("SELECT Id, Entity, Entity_Status FROM SP_Security_Instruments_Entities")
	if err != nil {
		log.Printf("getSecInstrumentsIdMap: %v", err)
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var d SecurityInstrumentsType
		if err := rows.Scan(&d.Id, &d.Entity, &d.EntityStatus); err == nil && d.EntityStatus {
			m[d.Id] = d
		}
	}
	return m
}

func (a *App) getEraIdMap() map[int]ERAEntities {
	m := make(map[int]ERAEntities)
	rows, err := a.DB.Query("SELECT Id, Entity, Entity_Status FROM SP_Exception_Restriction_Adverse_Entities")
	if err != nil {
		log.Printf("getEraIdMap: %v", err)
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var e ERAEntities
		if err := rows.Scan(&e.Id, &e.Entity, &e.EntityStatus); err == nil && e.EntityStatus {
			m[e.Id] = e
		}
	}
	return m
}

func (a *App) getTaxEntitiesIdMap() map[int]TaxEntities {
	m := make(map[int]TaxEntities)
	rows, err := a.DB.Query("SELECT Id, Entity, Entity_Status FROM SP_Tax_Entities")
	if err != nil {
		log.Printf("getTaxEntitiesIdMap: %v", err)
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var t TaxEntities
		if err := rows.Scan(&t.Id, &t.Entity, &t.EntityStatus); err == nil && t.EntityStatus {
			m[t.Id] = t
		}
	}
	return m
}

func (a *App) getChainOfTitleTypeIdMap() map[int]ChainOfTitleEntities {
	m := make(map[int]ChainOfTitleEntities)
	rows, err := a.DB.Query("SELECT Id, Entity, Entity_Status FROM SP_Chain_Of_Title_Entities")
	if err != nil {
		log.Printf("getChainOfTitleTypeIdMap: %v", err)
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var c ChainOfTitleEntities
		if err := rows.Scan(&c.Id, &c.Entity, &c.EntityStatus); err == nil && c.EntityStatus {
			m[c.Id] = c
		}
	}
	return m
}

func (a *App) getLienJudgementIdMap() map[int]LienJudgementEntities {
	m := make(map[int]LienJudgementEntities)
	rows, err := a.DB.Query("SELECT Id, Entity, Entity_Status FROM SP_Lien_Judgement_Entities")
	if err != nil {
		log.Printf("getLienJudgementIdMap: %v", err)
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var l LienJudgementEntities
		if err := rows.Scan(&l.Id, &l.Entity, &l.EntityStatus); err == nil && l.EntityStatus {
			m[l.Id] = l
		}
	}
	return m
}

func (a *App) getTaxAuthorityTypeMap() map[int]TaxAuthorityType {
	m := make(map[int]TaxAuthorityType)
	rows, err := a.DB.Query("SELECT Id, Name FROM SP_Tax_Authority_Types")
	if err != nil {
		log.Printf("getTaxAuthorityTypeMap: %v", err)
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var t TaxAuthorityType
		if err := rows.Scan(&t.Id, &t.Name); err == nil {
			m[t.Id] = t
		}
	}
	return m
}

func (a *App) getPropertyInterestTypes() map[int]PropertyInterestType {
	m := make(map[int]PropertyInterestType)
	rows, err := a.DB.Query("SELECT Id, Interest_Type_Name FROM Property_Interest_Types")
	if err != nil {
		log.Printf("getPropertyInterestTypes: %v", err)
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var p PropertyInterestType
		if err := rows.Scan(&p.Id, &p.InterestTypeName); err == nil {
			m[p.Id] = p
		}
	}
	return m
}

// getAllMaps is kept as a convenience aggregator (used by legacy call-sites).
func (a *App) getAllMaps() RbSectionTypeMaps {
	return RbSectionTypeMaps{
		DocTypesMap:         a.getDocTypeIdMap(),
		TaxSourcesMap:       a.getTaxSourceTypeIdMap(),
		TaxPaidStatusMap:    a.getTaxPaidStatusMap(),
		ChainTypesMap:       a.getChainTypesIdMap(),
		LienTypesMap:        a.getLienTypesIdMap(),
		PartyEntitiesMap:    a.getSPPartyIdMap(),
		SecInstEntitiesMap:  a.getSecInstrumentsIdMap(),
		ERAEntitiesMap:      a.getEraIdMap(),
		TaxEntitiesMap:      a.getTaxEntitiesIdMap(),
		ChainEntitiesMap:    a.getChainOfTitleTypeIdMap(),
		LienEntitiesMap:     a.getLienJudgementIdMap(),
		TaxAuthorityTypeMap: a.getTaxAuthorityTypeMap(),
		InterestTypeMap:     a.getPropertyInterestTypes(),
	}
}
