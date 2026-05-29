// Package rb — data optimization pipeline.
// Mirrors optimizeReportData() and all cleanse* functions from package_helper.js.
package rb

import (
	"sort"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"
)

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

// optimizeReportData is the Go equivalent of optimizeReportData() in package_helper.js (L54-L73).
// All 9 cleanse functions run concurrently via errgroup (mirrors Promise.all).
func optimizeReportData(payload *SearchPackagePayload) (*OptimizedReport, error) {
	// Filter "Does Not Apply" items before any cleansing.
	filterForApplies(payload)

	var (
		propertyDetails map[string]any
		searchParties   []map[string]any
		assessments     []map[string]any
		taxes           []map[string]any
		vesting         []map[string]any
		chain           []map[string]any
		security        []map[string]any
		liens           []map[string]any
		era             []map[string]any
	)

	g := new(errgroup.Group)

	g.Go(func() error { propertyDetails = cleanseProperty(payload); return nil })
	g.Go(func() error { searchParties = cleanseSearchParties(payload); return nil })
	g.Go(func() error { assessments = cleanseAssessments(payload); return nil })
	g.Go(func() error { taxes = cleanseTaxes(payload); return nil })
	g.Go(func() error { vesting = cleanseVesting(payload); return nil })
	g.Go(func() error { chain = cleanseChain(payload); return nil })
	g.Go(func() error { security = cleanseSecurity(payload); return nil })
	g.Go(func() error { liens = cleanseLiens(payload); return nil })
	g.Go(func() error { era = cleanseERA(payload); return nil })

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// finalizeData: filter each section to Status==1 (mirrors finalizeData() L75-L92)
	isActive := func(m map[string]any) bool { return getInt(m, "Status") == 1 }

	return &OptimizedReport{
		PropertyDetails:     propertyDetails,
		ReportDetails:       propertyDetails, // same reference as in JS
		SearchParties:       filterMaps(searchParties, isActive),
		Assessments:         filterMaps(assessments, isActive),
		Taxes:               filterMaps(taxes, isActive),
		VestingDeed:         filterMaps(vesting, isActive),
		ChainOfTitle:        filterMaps(chain, isActive),
		SecurityInstruments: filterMaps(security, isActive),
		LienAndJudgement:    filterMaps(liens, isActive),
		ERA:                 filterMaps(era, isActive),
		GeneralComments:     filterMaps(payload.BuilderData.SPGeneralComments, isActive),
	}, nil
}

func filterMaps(items []map[string]any, pred func(map[string]any) bool) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if pred(item) {
			out = append(out, item)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// filterForApplies — mirrors filterForApplies() in package_helper.js (L94-L127)
// Removes items where Applies==0 when Include_Does_Not_Apply is NOT set.
// ---------------------------------------------------------------------------

func filterForApplies(payload *SearchPackagePayload) {
	if payload.BuilderData.SearchPackage.IncludeDoesNotApply.Int64 != 0 {
		return
	}
	notDNA := func(m map[string]any) bool { return getInt(m, "Applies") != 0 }

	payload.BuilderData.SPAssessments = filterMaps(payload.BuilderData.SPAssessments, notDNA)
	payload.BuilderData.SPTaxes = filterMaps(payload.BuilderData.SPTaxes, notDNA)
	payload.BuilderData.SPChainOfTitlesVD = filterMaps(payload.BuilderData.SPChainOfTitlesVD, notDNA)
	payload.BuilderData.SPChainOfTitlesCT = filterMaps(payload.BuilderData.SPChainOfTitlesCT, notDNA)
	payload.BuilderData.SPSecurityInstruments = filterMaps(payload.BuilderData.SPSecurityInstruments, notDNA)
	payload.BuilderData.SPLienJudgement = filterMaps(payload.BuilderData.SPLienJudgement, notDNA)
	payload.BuilderData.SPExceptionRestrictionAdverse = filterMaps(payload.BuilderData.SPExceptionRestrictionAdverse, notDNA)

	// Parties are typed — filter in place
	filtered := make([]SPParty, 0, len(payload.BuilderData.SPParties))
	for _, p := range payload.BuilderData.SPParties {
		if p.Applies.Int64 != 0 {
			filtered = append(filtered, p)
		}
	}
	payload.BuilderData.SPParties = filtered
}

// ---------------------------------------------------------------------------
// cleanseProperty — mirrors cleanseProperty() in package_helper.js (L135-L186)
// Converts the typed SearchPackage struct to map[string]any and adds display fields.
// ---------------------------------------------------------------------------

func cleanseProperty(payload *SearchPackagePayload) map[string]any {
	sp := payload.BuilderData.SearchPackage
	m := map[string]any{
		"Id":                     sp.ID,
		"Order_ID":               sp.OrderID.Int64,
		"VD_Manual_Sort":         sp.VDManualSort.Int64,
		"CT_Manual_Sort":         sp.CTManualSort.Int64,
		"SI_Manual_Sort":         sp.SIManualSort.Int64,
		"LJ_Manual_Sort":         sp.LJManualSort.Int64,
		"ER_Manual_Sort":         sp.ERManualSort.Int64,
		"PUD":                    sp.PUD.Int64,
		"Parcel":                 sp.Parcel.String,
		"Registry":               sp.Registry.Int64,
		"Interest_Type_Id":       sp.InterestTypeID.Int64,
		"Include_Does_Not_Apply": sp.IncludeDoesNotApply.Int64,
	}

	// Estate label
	m["estateLabel"] = getEstateType(int(sp.InterestTypeID.Int64), payload.PrepData.InterestTypes)

	// Parcel text with registry suffix
	if p := sp.Parcel.String; p != "" {
		suffix := ""
		switch sp.Registry.Int64 {
		case 1:
			suffix = "(Recorded Land)"
		case 2:
			suffix = "(Land Court)"
		case 3:
			suffix = "(Others)"
		}
		m["ParcelText"] = p + suffix
	} else {
		m["ParcelText"] = "N/A"
	}

	// Document presence
	docList := payload.BuilderData.SPDocuments
	spDocInReport := payload.OrderData.SPDocInReport.Int64 != 0

	rprtTypeID := docTypeID(payload.PrepData.DocTypes, "Additional Document")
	m["rprtDocPresent"] = anyDocOfType(docList, rprtTypeID) && spDocInReport

	propTypeID := docTypeID(payload.PrepData.DocTypes, "Property")
	m["propDocPresent"] = anyDocOfType(docList, propTypeID) && spDocInReport

	return m
}

func getEstateType(id int, types []PropertyInterestType) string {
	for _, t := range types {
		if t.Id == id {
			return t.InterestTypeName
		}
	}
	return "N/A"
}

func docTypeID(docTypes []DocumentType, name string) int {
	for _, dt := range docTypes {
		if dt.Type == name {
			return dt.Id
		}
	}
	return 0
}

// anyDocOfType checks if any doc in the list has the given Type_ID.
func anyDocOfType(docList []map[string]any, typeID int) bool {
	for _, doc := range docList {
		if getInt(doc, "Type_ID") == typeID {
			return true
		}
	}
	return false
}

// anyDocForEntity checks if any doc matches both Type_ID and Entity_ID.
func anyDocForEntity(docList []map[string]any, typeID, entityID int) bool {
	for _, doc := range docList {
		if getInt(doc, "Type_ID") == typeID && getInt(doc, "Entity_ID") == entityID {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// cleanseSearchParties — mirrors cleanseSearchParties() (package_helper.js L188-L240)
// ---------------------------------------------------------------------------

func cleanseSearchParties(payload *SearchPackagePayload) []map[string]any {
	docList := payload.BuilderData.SPDocuments
	typeID := docTypeID(payload.PrepData.DocTypes, "Search Party")
	titleDocPresent := anyDocOfType(docList, typeID) && payload.OrderData.SPDocInReport.Int64 != 0

	result := make([]map[string]any, 0, len(payload.BuilderData.SPParties))
	for _, party := range payload.BuilderData.SPParties {
		entityID := int(party.EntityID.Int64)

		// Entity label
		entityLabel := "N/A"
		for _, pe := range payload.PrepData.PartyEntities {
			if pe.Id == entityID {
				entityLabel = pe.Entity
				break
			}
		}

		// Full name (mirrors the spName building logic)
		spName := "-"
		addPart := func(s string) {
			s = strings.TrimSpace(s)
			if s == "" {
				return
			}
			if spName == "-" {
				spName = s + " "
			} else {
				spName += s + " "
			}
		}
		addPart(party.FirstBusinessName.String)
		addPart(party.MiddleName.String)
		addPart(party.LastName.String)

		// "Also Searched" alias string (IND entities only, non-legal aliases)
		var nameSearched string
		if entityID == 1 || entityID == 2 {
			for _, alias := range party.Alias {
				a := strings.TrimSpace(alias.Alias.String)
				if a != "" && alias.IsLegal.Int64 == 0 {
					if nameSearched != "" {
						nameSearched += ", " + a
					} else {
						nameSearched = a
					}
				}
			}
		}

		docPresent := anyDocForEntity(docList, typeID, party.ID) && payload.OrderData.SPDocInReport.Int64 != 0

		m := map[string]any{
			"Id":                  party.ID,
			"Sp_Id":               party.SpID,
			"Entity_ID":           entityID,
			"First_Business_Name": party.FirstBusinessName.String,
			"Middle_Name":         party.MiddleName.String,
			"Last_Name":           party.LastName.String,
			"Status":              party.Status.Int64,
			"Applies":             party.Applies.Int64,
			"Entity_Label":        entityLabel,
			"spName":              strings.TrimSpace(spName),
			"titleDocPresent":     titleDocPresent,
			"docPresent":          docPresent,
		}
		if nameSearched != "" {
			m["spAlias"] = "Also Searched: " + nameSearched
		}
		result = append(result, m)
	}
	return result
}

// ---------------------------------------------------------------------------
// cleanseAssessments — mirrors cleanseAssessments() (package_helper.js L242-L276)
// ---------------------------------------------------------------------------

func cleanseAssessments(payload *SearchPackagePayload) []map[string]any {
	data := payload.BuilderData.SPAssessments
	docList := payload.BuilderData.SPDocuments
	typeID := docTypeID(payload.PrepData.DocTypes, "Assessment")
	titleDocPresent := anyDocOfType(docList, typeID) && payload.OrderData.SPDocInReport.Int64 != 0

	for _, elm := range data {
		elm["titleDocPresent"] = titleDocPresent
		elm["docPresent"] = anyDocForEntity(docList, typeID, getInt(elm, "Id")) && payload.OrderData.SPDocInReport.Int64 != 0
		elm["landVal"] = dollarFormat(getFloat(elm, "Land"))
		elm["buildVal"] = dollarFormat(getFloat(elm, "Building"))
		elm["extraVal"] = dollarFormat(getFloat(elm, "Extras"))
		elm["totVal"] = dollarFormat(getFloat(elm, "Total"))
	}
	return data
}

// ---------------------------------------------------------------------------
// cleanseTaxes — mirrors cleanseTaxes() (package_helper.js L278-L319)
// ---------------------------------------------------------------------------

func cleanseTaxes(payload *SearchPackagePayload) []map[string]any {
	data := payload.BuilderData.SPTaxes
	docList := payload.BuilderData.SPDocuments
	typeID := docTypeID(payload.PrepData.DocTypes, "Tax")
	titleDocPresent := anyDocOfType(docList, typeID) && payload.OrderData.SPDocInReport.Int64 != 0

	// Filter invalid items (where all paid statuses are "Not Applicable" = 7)
	valid := make([]map[string]any, 0, len(data))
	for _, elm := range data {
		if isValidTax(elm) {
			valid = append(valid, elm)
		}
	}

	for _, elm := range valid {
		elm["titleDocPresent"] = titleDocPresent
		elm["docPresent"] = anyDocForEntity(docList, typeID, getInt(elm, "Id")) && payload.OrderData.SPDocInReport.Int64 != 0

		// Authority type label
		taxTypeID := getInt(elm, "Tax_Type_ID")
		for _, at := range payload.PrepData.TaxAuthorityType {
			if at.Id == taxTypeID {
				elm["prtTaxAuthorityType"] = at.Name
				break
			}
		}

		switch getInt(elm, "Entity_ID") {
		case 1, 4:
			setAnnualTax(elm, payload.PrepData.TaxSources, payload.PrepData.TaxStatus)
		case 2:
			setSemiAnnualTax(elm, payload.PrepData.TaxSources, payload.PrepData.TaxStatus)
		case 3:
			setQuarterlyTax(elm, payload.PrepData.TaxSources, payload.PrepData.TaxStatus)
		}
	}
	return valid
}

const notApplicableTaxID = 7

func isValidTax(elm map[string]any) bool {
	switch getInt(elm, "Entity_ID") {
	case 1, 4:
		return getInt(elm, "Annual_Paid_Status_Type_ID") != notApplicableTaxID
	case 2:
		return getInt(elm, "Fst_Semi_Annual_Paid_Status_Type_ID") != notApplicableTaxID ||
			getInt(elm, "Snd_Semi_Annual_Paid_Status_Type_ID") != notApplicableTaxID
	case 3:
		for _, k := range []string{"Fst", "Snd", "Thrd", "Frth"} {
			if getInt(elm, k+"_Quarter_Paid_Status_Type_ID") != notApplicableTaxID {
				return true
			}
		}
	}
	return false
}

func taxPaidLabel(id int, statuses []TaxPaidStatus) string {
	for _, s := range statuses {
		if s.Id == id {
			return strings.ToUpper(s.StatusType)
		}
	}
	return "N/A"
}

func taxSourceLabel(id int, sources []TaxType) string {
	for _, s := range sources {
		if s.Id == id {
			return s.SourceType
		}
	}
	return "N/A"
}

func setAnnualTax(elm map[string]any, sources []TaxType, statuses []TaxPaidStatus) {
	year := getStr(elm, "Year")
	suffix := map[int]string{1: "Annual", 4: "Special Assessment"}[getInt(elm, "Entity_ID")]
	taxYear := ""
	if year != "" {
		taxYear = year + "-" + suffix
	}
	elm["taxYear"] = taxYear
	elm["prtDtLbl"] = "Date"
	elm["prtTaxStat"] = taxPaidLabel(getInt(elm, "Annual_Paid_Status_Type_ID"), statuses)
	elm["prtTaxAmt"] = dollarFormat(getFloat(elm, "Annual_Amount"))
	elm["prtPaidDate"] = formatDate(getStr(elm, "Annual_Paid_Date"))
	elm["prtTxDelinq"] = delinqLabel(getInt(elm, "Prior_Delinquencies"))
	elm["prtSource"] = taxSourceLabel(getInt(elm, "Source_Type_ID"), sources)
}

func setSemiAnnualTax(elm map[string]any, sources []TaxType, statuses []TaxPaidStatus) {
	elm["taxYear"] = getStr(elm, "Year")
	if getFloat(elm, "Fst_Semi_Annual_Amount") >= 0 {
		elm["taxPeriod1"] = "1st Semi-Annual"
		elm["prtDtLbl"] = "Date"
		elm["prtTaxStat"] = taxPaidLabel(getInt(elm, "Fst_Semi_Annual_Paid_Status_Type_ID"), statuses)
		elm["prtTaxAmt"] = dollarFormat(getFloat(elm, "Fst_Semi_Annual_Amount"))
		elm["prtPaidDate"] = formatDate(getStr(elm, "Fst_Semi_Annual_Paid_Date"))
	}
	if getFloat(elm, "Snd_Semi_Annual_Amount") >= 0 {
		elm["taxPeriod2"] = "2nd Semi-Annual"
		elm["prtDtLbl2"] = "Date"
		elm["prtTaxStat2"] = taxPaidLabel(getInt(elm, "Snd_Semi_Annual_Paid_Status_Type_ID"), statuses)
		elm["prtTaxAmt2"] = dollarFormat(getFloat(elm, "Snd_Semi_Annual_Amount"))
		elm["prtPaidDate2"] = formatDate(getStr(elm, "Snd_Semi_Annual_Paid_Date"))
	}
	elm["prtTxDelinq"] = delinqLabel(getInt(elm, "Prior_Delinquencies"))
	elm["prtSource"] = taxSourceLabel(getInt(elm, "Source_Type_ID"), sources)
}

func setQuarterlyTax(elm map[string]any, sources []TaxType, statuses []TaxPaidStatus) {
	elm["taxYear"] = getStr(elm, "Year")
	type quarter struct {
		amt, stat, date, label, sfx string
	}
	quarters := []quarter{
		{"Fst_Quarter_Amount", "Fst_Quarter_Paid_Status_Type_ID", "Fst_Quarter_Paid_Date", "1st Quarter", ""},
		{"Snd_Quarter_Amount", "Snd_Quarter_Paid_Status_Type_ID", "Snd_Quarter_Paid_Date", "2nd Quarter", "2"},
		{"Thrd_Quarter_Amount", "Thrd_Quarter_Paid_Status_Type_ID", "Thrd_Quarter_Paid_Date", "3rd Quarter", "3"},
		{"Frth_Quarter_Amount", "Frth_Quarter_Paid_Status_Type_ID", "Frth_Quarter_Paid_Date", "4th Quarter", "4"},
	}
	for _, q := range quarters {
		if getFloat(elm, q.amt) >= 0 {
			elm["taxPeriod"+q.sfx] = q.label
			elm["prtDtLbl"+q.sfx] = "Date"
			elm["prtTaxStat"+q.sfx] = taxPaidLabel(getInt(elm, q.stat), statuses)
			elm["prtTaxAmt"+q.sfx] = dollarFormat(getFloat(elm, q.amt))
			elm["prtPaidDate"+q.sfx] = formatDate(getStr(elm, q.date))
		}
	}
	elm["prtTxDelinq"] = delinqLabel(getInt(elm, "Prior_Delinquencies"))
	elm["prtSource"] = taxSourceLabel(getInt(elm, "Source_Type_ID"), sources)
}

// ---------------------------------------------------------------------------
// cleanseVesting / cleanseChain / shared cleanseVestAndChain
// Mirrors cleanseVesting(), cleanseChain(), cleanseVestAndChain() (L491-L605)
// ---------------------------------------------------------------------------

func cleanseVesting(payload *SearchPackagePayload) []map[string]any {
	data := payload.BuilderData.SPChainOfTitlesVD
	if payload.BuilderData.SearchPackage.VDManualSort.Int64 == 0 {
		sortByRecDate(data, false)
		for i, item := range data {
			item["Sort_Order"] = i
		}
	}
	return cleanseVestAndChain(data, docTypeID(payload.PrepData.DocTypes, "Vesting Deed"), payload)
}

func cleanseChain(payload *SearchPackagePayload) []map[string]any {
	data := payload.BuilderData.SPChainOfTitlesCT
	if payload.BuilderData.SearchPackage.CTManualSort.Int64 == 0 {
		sortByRecDate(data, false)
		for i, item := range data {
			item["Sort_Order"] = i
		}
	}
	return cleanseVestAndChain(data, docTypeID(payload.PrepData.DocTypes, "Chain of Title"), payload)
}

func cleanseVestAndChain(data []map[string]any, typeID int, payload *SearchPackagePayload) []map[string]any {
	docList := payload.BuilderData.SPDocuments
	titleDocPresent := anyDocOfType(docList, typeID) && payload.OrderData.SPDocInReport.Int64 != 0

	for _, elm := range data {
		entityID := getInt(elm, "Entity_ID")

		// Deed entity label
		deedEntityLabel := ""
		for _, ce := range payload.PrepData.ChainEntities {
			if ce.Id == entityID {
				deedEntityLabel = ce.Entity
				break
			}
		}

		elm["titleDocPresent"] = titleDocPresent
		elm["prtDeedDateLbl"] = "Dated"
		elm["prtFromLbl"] = "Grantor"
		elm["prtToLbl"] = "Grantee"
		elm["prtFromData"] = getStr(elm, "Grantor")
		elm["prtToData"] = getStr(elm, "Grantee")
		elm["deedEntityLabel"] = deedEntityLabel
		elm["prtDeedDate"] = formatDate(getStr(elm, "Dated_Date"))

		if entityID == 2 {
			elm["prtDeedDateLbl"] = "Date of Death"
			elm["prtFromLbl"] = "Estate of"
			elm["prtToLbl"] = "Beneficiaries"
			elm["prtFromData"] = getStr(elm, "Estate_Of")
			elm["prtToData"] = getStr(elm, "Beneficiaries")
			elm["prtDeedDate"] = formatDate(getStr(elm, "Date_Of_Death"))
		}

		// Chain instrument type
		instrTypeID := getInt(elm, "Instrument_Type_ID")
		for _, ct := range payload.PrepData.ChainTypes {
			if ct.Id == instrTypeID {
				if ct.Id == 12 && strings.TrimSpace(getStr(elm, "Others")) != "" {
					elm["prtDeedType"] = strings.TrimSpace(getStr(elm, "Others"))
				} else {
					elm["prtDeedType"] = ct.Type
				}
				break
			}
		}

		bi := bookInstInfo(elm)
		elm["prtInstNum"] = bi.value
		elm["prtInstNumLabel"] = bi.label
		elm["prtRecDate"] = formatDate(getStr(elm, "Rec_Date"))
		elm["prtCsndrtnAmt"] = dollarFormat(getFloat(elm, "Consideration"))
		elm["docPresent"] = anyDocForEntity(docList, typeID, getInt(elm, "Id")) && titleDocPresent
	}
	return data
}

// ---------------------------------------------------------------------------
// cleanseSecurity — mirrors cleanseSecurity() (package_helper.js L607-L680)
// ---------------------------------------------------------------------------

func cleanseSecurity(payload *SearchPackagePayload) []map[string]any {
	data := payload.BuilderData.SPSecurityInstruments
	if payload.BuilderData.SearchPackage.SIManualSort.Int64 == 0 {
		sortByRecDate(data, true)
		for i, item := range data {
			item["Sort_Order"] = i
		}
	}

	docList := payload.BuilderData.SPDocuments
	typeID := docTypeID(payload.PrepData.DocTypes, "Security Instrument")
	titleDocPresent := anyDocOfType(docList, typeID) && payload.OrderData.SPDocInReport.Int64 != 0
	mortgageEntities := map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true, 12: true, 13: true, 14: true}
	pudYN := "No"
	if payload.BuilderData.SearchPackage.PUD.Int64 != 0 {
		pudYN = "Yes"
	}

	for _, elm := range data {
		entityID := getInt(elm, "Entity_ID")
		entityLabel := "N/A"
		for _, se := range payload.PrepData.SecInstEntities {
			if se.Id == entityID {
				entityLabel = se.Entity
				break
			}
		}

		elm["Entity_Label"] = entityLabel
		elm["titleDocPresent"] = titleDocPresent
		elm["docPresent"] = anyDocForEntity(docList, typeID, getInt(elm, "Id")) && payload.OrderData.SPDocInReport.Int64 != 0

		bi := bookInstInfo(elm)
		elm["prtInstNum"] = bi.value
		elm["prtInstNumLabel"] = bi.label
		elm["prtLbl2"], elm["prtLbl3"], elm["prtLbl4"], elm["prtLbl5"] = "", "Recorded", "", ""
		elm["prtData1"] = formatDate(getStr(elm, "Dated_Date"))
		elm["prtData2"], elm["prtData3"] = "", formatDate(getStr(elm, "Rec_Date"))
		elm["prtData4"], elm["prtData5"] = "", ""

		if mortgageEntities[entityID] {
			elm["prtLbl2"], elm["prtLbl3"], elm["prtLbl4"], elm["prtLbl5"] = "Recorded", "Maturity", "PUD", "Amount"
			elm["prtData2"] = formatDate(getStr(elm, "Rec_Date"))
			elm["prtData3"] = formatDate(getStr(elm, "Maturity_Date"))
			elm["prtData4"] = pudYN
			elm["prtData5"] = dollarFormat(getFloat(elm, "Amount"))
		}
	}
	return data
}

// ---------------------------------------------------------------------------
// cleanseLiens — mirrors cleanseLiens() (package_helper.js L682-L735)
// ---------------------------------------------------------------------------

func cleanseLiens(payload *SearchPackagePayload) []map[string]any {
	data := payload.BuilderData.SPLienJudgement
	if payload.BuilderData.SearchPackage.LJManualSort.Int64 == 0 {
		sortByRecDate(data, true)
		for i, item := range data {
			item["Sort_Order"] = i
		}
	}

	docList := payload.BuilderData.SPDocuments
	typeID := docTypeID(payload.PrepData.DocTypes, "Lien & Judgement")
	titleDocPresent := anyDocOfType(docList, typeID) && payload.OrderData.SPDocInReport.Int64 != 0

	for _, elm := range data {
		elm["headerText"] = lienType(elm, payload.PrepData.LienTypes)
		elm["titleDocPresent"] = titleDocPresent
		elm["docPresent"] = anyDocForEntity(docList, typeID, getInt(elm, "Id")) && payload.OrderData.SPDocInReport.Int64 != 0

		bi := bookInstInfo(elm)
		elm["prtInstNum"] = bi.value
		elm["prtInstNumLabel"] = bi.label
		elm["prtLienDate"] = formatDate(getStr(elm, "Dated_Date"))
		elm["prtRecDate"] = formatDate(getStr(elm, "Rec_Date"))
		elm["prtLienAmt"] = dollarFormat(getFloat(elm, "Amount"))
		elm["prtHdr1"], elm["prtHdr2"] = "Debtor", "Holder"
		switch getInt(elm, "Entity_ID") {
		case 2:
			elm["prtHdr1"], elm["prtHdr2"] = "Defendant", "In Favor Of"
		case 3:
			elm["prtHdr1"], elm["prtHdr2"] = "In Favor Of", ""
		}
	}
	return data
}

func lienType(elm map[string]any, lienTypes []LienJudgementType) string {
	typeID := getInt(elm, "Type_ID")
	customIDs := map[int]bool{53: true, 64: true}

	if customIDs[typeID] {
		if t := strings.TrimSpace(getStr(elm, "Type")); t != "" {
			lines := strings.Split(t, "\n")
			for i := len(lines) - 1; i >= 0; i-- {
				if l := strings.TrimSpace(lines[i]); l != "" {
					return l
				}
			}
		}
	}
	if typeID != 0 {
		for _, lt := range lienTypes {
			if lt.Id == typeID {
				return lt.Type
			}
		}
	}
	return "Other"
}

// ---------------------------------------------------------------------------
// cleanseERA — mirrors cleanseERA() (package_helper.js L759-L801)
// ---------------------------------------------------------------------------

func cleanseERA(payload *SearchPackagePayload) []map[string]any {
	data := payload.BuilderData.SPExceptionRestrictionAdverse
	if payload.BuilderData.SearchPackage.ERManualSort.Int64 == 0 {
		sortByRecDate(data, true)
		for i, item := range data {
			item["Sort_Order"] = i
		}
	}

	docList := payload.BuilderData.SPDocuments
	typeID := docTypeID(payload.PrepData.DocTypes, "ERA")
	titleDocPresent := anyDocOfType(docList, typeID) && payload.OrderData.SPDocInReport.Int64 != 0

	for _, elm := range data {
		entityID := getInt(elm, "Entity_ID")
		entityLabel := "N/A"
		for _, ee := range payload.PrepData.ERAEntities {
			if ee.Id == entityID {
				entityLabel = ee.Entity
				break
			}
		}

		elm["Entity_Label"] = entityLabel
		elm["titleDocPresent"] = titleDocPresent
		elm["docPresent"] = anyDocForEntity(docList, typeID, getInt(elm, "Id")) && payload.OrderData.SPDocInReport.Int64 != 0

		bi := bookInstInfo(elm)
		elm["prtInstNum"] = bi.value
		elm["prtInstNumLabel"] = bi.label
		elm["prtERADate"] = formatDate(getStr(elm, "Dated_Date"))
		elm["prtRecDate"] = formatDate(getStr(elm, "Rec_Date"))
		elm["prtERAGrantor"] = getStr(elm, "Grantor")
		elm["prtERAGrantee"] = getStr(elm, "Grantee")
	}
	return data
}

// ---------------------------------------------------------------------------
// Book/Instrument info — mirrors getBookOrInstrumentLabelAndValue() (L950-L977)
// ---------------------------------------------------------------------------

type bookInst struct{ label, value string }

func bookInstInfo(elm map[string]any) bookInst {
	if num := strings.TrimSpace(getStr(elm, "Instrument_Num")); num != "" {
		return bookInst{"Instrument #", num}
	}
	bookVal := ""
	for _, k := range []string{"Book_Num", "Book", "Book_Case"} {
		if v := strings.TrimSpace(getStr(elm, k)); v != "" {
			bookVal = v
			break
		}
	}
	pageVal := ""
	for _, k := range []string{"Page", "PG_Case"} {
		if v := strings.TrimSpace(getStr(elm, k)); v != "" {
			pageVal = v
			break
		}
	}
	if bookVal != "" {
		if pageVal != "" {
			return bookInst{"Bk/Pg", bookVal + "/" + pageVal}
		}
		return bookInst{"Book", bookVal}
	}
	if pageVal != "" {
		return bookInst{"Page", pageVal}
	}
	return bookInst{"Instrument #", ""}
}

// ---------------------------------------------------------------------------
// Sorting — mirrors customSortingFunction() (package_helper.js L902-L910)
// ---------------------------------------------------------------------------

func sortByRecDate(data []map[string]any, ascending bool) {
	sort.SliceStable(data, func(i, j int) bool {
		// Primary: Rec_Date
		if d := cmpRecDate(data[i], data[j], ascending); d != 0 {
			return d < 0
		}
		// Secondary: Instrument_Num
		if d := cmpIntField("Instrument_Num", data[i], data[j], ascending); d != 0 {
			return d < 0
		}
		// Tertiary: Book number
		if d := cmpBookNum(data[i], data[j], ascending); d != 0 {
			return d < 0
		}
		// Quaternary: Page
		return cmpIntField("Page", data[i], data[j], ascending) < 0
	})
}

func cmpRecDate(a, b map[string]any, ascending bool) int {
	ta, okA := parseAnyDate(getStr(a, "Rec_Date"))
	tb, okB := parseAnyDate(getStr(b, "Rec_Date"))
	if !okA && !okB {
		return 0
	}
	if !okA {
		return 1
	}
	if !okB {
		return -1
	}
	diff := tb.Unix() - ta.Unix()
	if ascending {
		diff = -diff
	}
	if diff > 0 {
		return 1
	} else if diff < 0 {
		return -1
	}
	return 0
}

func cmpIntField(key string, a, b map[string]any, ascending bool) int {
	diff := getInt(b, key) - getInt(a, key)
	if ascending {
		diff = -diff
	}
	return diff
}

func cmpBookNum(a, b map[string]any, ascending bool) int {
	bookNum := func(m map[string]any) int {
		for _, k := range []string{"Book_Num", "Book", "Book_Case"} {
			if v := strings.TrimSpace(getStr(m, k)); v != "" {
				n, _ := strconv.Atoi(v)
				return n
			}
		}
		return 0
	}
	diff := bookNum(b) - bookNum(a)
	if ascending {
		diff = -diff
	}
	return diff
}

// ---------------------------------------------------------------------------
// sortDocuments — mirrors sortDocuments() (package_helper.js L809-L888)
// ---------------------------------------------------------------------------

func sortDocuments(
	docList []map[string]any,
	docTypes []DocumentType,
	report *OptimizedReport,
) []SortedDoc {
	var result []SortedDoc

	appendSorted := func(typeID, entityID, idx int) {
		result = append(result, getSortedDocs(docList, typeID, entityID, idx)...)
	}

	// Property docs (no entityID filter)
	appendSorted(docTypeID(docTypes, "Property"), 0, 0)

	sectionTypes := []struct {
		items   []map[string]any
		docType string
	}{
		{report.Assessments, "Assessment"},
		{report.Taxes, "Tax"},
		{report.VestingDeed, "Vesting Deed"},
		{report.ChainOfTitle, "Chain of Title"},
		{report.SecurityInstruments, "Security Instrument"},
		{report.LienAndJudgement, "Lien & Judgement"},
		{report.ERA, "ERA"},
		{report.SearchParties, "Search Party"},
	}

	for _, sec := range sectionTypes {
		tid := docTypeID(docTypes, sec.docType)
		for idx, item := range sec.items {
			appendSorted(tid, getInt(item, "Id"), idx)
		}
	}

	// Additional documents
	appendSorted(docTypeID(docTypes, "Additional Document"), 0, 0)

	return result
}

func getSortedDocs(docList []map[string]any, typeID, entityID, idx int) []SortedDoc {
	var filtered []map[string]any
	for _, doc := range docList {
		if getInt(doc, "Type_ID") == typeID &&
			getInt(doc, "Entity_ID") == entityID &&
			getInt(doc, "Status") == 1 {
			filtered = append(filtered, doc)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		return getInt(filtered[i], "Sort_Order") < getInt(filtered[j], "Sort_Order")
	})
	result := make([]SortedDoc, 0, len(filtered))
	for _, doc := range filtered {
		result = append(result, SortedDoc{Path: doc, Idx: idx})
	}
	return result
}
