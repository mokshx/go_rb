package rb

import (
	"database/sql"
)

// ---------------------------------------------------------------------------
// Request / context
// ---------------------------------------------------------------------------

// CreateSearchPackageRequest mirrors the Express req object fields consumed by
// the createSearchPackage flow.
type CreateSearchPackageRequest struct {
	OrderID     string
	BuilderID   string
	AdminUserID string
	IsPreview   bool
}

// ---------------------------------------------------------------------------
// Order_Property_User_View row
// ---------------------------------------------------------------------------

type OrderPropertyData struct {
	AbstractorUserFirstName sql.NullString `db:"Abstractor_User_First_Name"`
	AbstractorUserFullName  sql.NullString `db:"Abstractor_User_Full_Name"`
	AbstractorUserLastName  sql.NullString `db:"Abstractor_User_Last_Name"`
	AbstractorUserRole      sql.NullInt64  `db:"Abstractor_User_Role"`
	AbstractorUserStatus    sql.NullInt64  `db:"Abstractor_User_Status"`
	AbstratorID             sql.NullString `db:"Abstrator_ID"`
	AbstratorUserID         sql.NullString `db:"Abstrator_User_ID"`

	AdminUserFirstName sql.NullString `db:"Admin_User_First_Name"`
	AdminUserFullName  sql.NullString `db:"Admin_User_Full_Name"`
	AdminUserLastName  sql.NullString `db:"Admin_User_Last_Name"`
	AdminUserRole      sql.NullInt64  `db:"Admin_User_Role"`
	AdminUserStatus    sql.NullInt64  `db:"Admin_User_Status"`
	AdminUserType      sql.NullInt64  `db:"Admin_User_Type"`

	BorrowerSSN                     sql.NullString `db:"Borrower_SSN"`
	CoBorrowerPropertyAddress1      sql.NullString `db:"Co_Borrower_Property_Address_1"`
	CoBorrowerPropertyAddress2      sql.NullString `db:"Co_Borrower_Property_Address_2"`
	CoBorrowerPropertyCity          sql.NullString `db:"Co_Borrower_Property_City"`
	CoBorrowerPropertyCreationDate  sql.NullString `db:"Co_Borrower_Property_Creation_Date"`
	CoBorrowerPropertyFirstName     sql.NullString `db:"Co_Borrower_Property_First_Name"`
	CoBorrowerPropertyID            sql.NullString `db:"Co_Borrower_Property_ID"`
	CoBorrowerPropertyLastName      sql.NullString `db:"Co_Borrower_Property_Last_Name"`
	CoBorrowerPropertyLatitude      sql.NullString `db:"Co_Borrower_Property_Latitude"`
	CoBorrowerPropertyLongitude     sql.NullString `db:"Co_Borrower_Property_Longitude"`
	CoBorrowerPropertyState         sql.NullString `db:"Co_Borrower_Property_State"`
	CoBorrowerPropertyStateAbbr     sql.NullString `db:"Co_Borrower_Property_State_Abbr"`
	CoBorrowerPropertyStatus        sql.NullInt64  `db:"Co_Borrower_Property_Status"`
	CoBorrowerPropertyStatusMessage sql.NullString `db:"Co_Borrower_Property_Status_Message"`
	CoBorrowerPropertyZipCode       sql.NullString `db:"Co_Borrower_Property_ZipCode"`
	CoBorrowerSSN                   sql.NullString `db:"Co_Borrower_SSN"`

	CompanyID           sql.NullString `db:"Company_ID"`
	CompanyName         sql.NullString `db:"Company_Name"`
	CustomerID          sql.NullString `db:"Customer_ID"`
	DemoFlag            sql.NullInt64  `db:"Demo_Flag"`
	FileID              sql.NullString `db:"File_ID"`
	FrgnCompanyID       sql.NullString `db:"Frgn_Company_Id"`
	FrgnInternalOrderID sql.NullString `db:"Frgn_Internal_Order_Id"`
	LatestNote          sql.NullString `db:"Latest_Note"`
	LoanID              sql.NullString `db:"Loan_ID"`
	Logo                sql.NullString `db:"Logo"`

	OrderAddressManualFlag   sql.NullInt64  `db:"Order_Address_Manual_Flag"`
	OrderAdminID             sql.NullString `db:"Order_Admin_ID"`
	OrderAssignedDate        sql.NullString `db:"Order_Assigned_Date"`
	OrderAutomated           sql.NullInt64  `db:"Order_Automated"`
	OrderBook                sql.NullString `db:"Order_Book"`
	OrderBuyer               sql.NullString `db:"Order_Buyer"`
	OrderCancellationDate    sql.NullString `db:"Order_Cancellation_Date"`
	OrderCoBuyer             sql.NullString `db:"Order_Co_Buyer"`
	OrderCoSeller            sql.NullString `db:"Order_Co_Seller"`
	OrderCompletionDate      sql.NullString `db:"Order_Completion_Date"`
	OrderCreationDate        sql.NullString `db:"Order_Creation_Date"`
	OrderCustomPrice         sql.NullString `db:"Order_Custom_Price"`
	OrderCustomPriceTax      sql.NullString `db:"Order_Custom_Price_Tax"`
	OrderCustomerFileNumber  sql.NullString `db:"Order_Customer_File_Number"`
	OrderETA                 sql.NullString `db:"Order_ETA"`
	OrderETARange            sql.NullString `db:"Order_ETA_Range"`
	OrderEscalated           sql.NullInt64  `db:"Order_Escalated"`
	OrderEstimatedTime       sql.NullString `db:"Order_Estimated_Time"`
	OrderFinalPrice          sql.NullString `db:"Order_Final_Price"`
	OrderFinalPriceTax       sql.NullString `db:"Order_Final_Price_Tax"`
	OrderID                  sql.NullInt64  `db:"Order_ID"`
	OrderInternalDueTime     sql.NullString `db:"Order_Internal_Due_Time"`
	OrderLoan                sql.NullString `db:"Order_Loan"`
	OrderMaxTat              sql.NullInt64  `db:"Order_Max_Tat"`
	OrderMinTat              sql.NullInt64  `db:"Order_Min_Tat"`
	OrderModificationDate    sql.NullString `db:"Order_Modification_Date"`
	OrderMortgageAmount      sql.NullString `db:"Order_Mortgage_Amount"`
	OrderMortgageDate        sql.NullString `db:"Order_Mortgage_Date"`
	OrderOriginalETA         sql.NullString `db:"Order_Original_ETA"`
	OrderPage                sql.NullString `db:"Order_Page"`
	OrderParcel              sql.NullString `db:"Order_Parcel"`
	OrderPausedFlag          sql.NullInt64  `db:"Order_Paused_Flag"`
	OrderPredictionResult    sql.NullString `db:"Order_Prediction_Result"`
	OrderPriority            sql.NullInt64  `db:"Order_Priority"`
	OrderProductFlag         sql.NullInt64  `db:"Order_Product_Flag"`
	OrderProductStatus       sql.NullInt64  `db:"Order_Product_Status"`
	OrderPurpose             sql.NullString `db:"Order_Purpose"`
	OrderQualiaInternalID    sql.NullString `db:"Order_Qualia_Internal_ID"`
	OrderReportedErrorStatus sql.NullString `db:"Order_Reported_Error_Status"`
	OrderRequestedDate       sql.NullString `db:"Order_Requested_Date"`
	OrderSeller              sql.NullString `db:"Order_Seller"`
	OrderSentBackToScreen    sql.NullInt64  `db:"Order_Sent_Back_To_Screen"`
	OrderStatus              sql.NullInt64  `db:"Order_Status"`
	OrderStatusMessages      sql.NullString `db:"Order_Status_Messages"`
	OrderStripeChargeID      sql.NullString `db:"Order_Stripe_Charge_ID"`
	OrderSubdivision         sql.NullString `db:"Order_Subdivision"`
	OrderTags                sql.NullString `db:"Order_Tags"`
	OrderTagsFormatted       sql.NullString `db:"Order_Tags_Formatted"`

	OrganizationDescription sql.NullString `db:"Organization_Description"`
	OrganizationID          sql.NullString `db:"Organization_ID"`
	OrganizationName        sql.NullString `db:"Organization_Name"`
	OrganizationStatus      sql.NullInt64  `db:"Organization_Status"`

	ParsedPropertyAddress1 sql.NullString `db:"Parsed_Property_Address_1"`
	ProductDescription     sql.NullString `db:"Product_Description"`
	ProductID              sql.NullInt64  `db:"Product_ID"`
	ProductType            sql.NullString `db:"Product_Type"`

	PropertyAddress1      sql.NullString `db:"Property_Address_1"`
	PropertyAddress2      sql.NullString `db:"Property_Address_2"`
	PropertyCity          sql.NullString `db:"Property_City"`
	PropertyCounty        sql.NullString `db:"Property_County"`
	PropertyCreationDate  sql.NullString `db:"Property_Creation_Date"`
	PropertyFirstName     sql.NullString `db:"Property_First_Name"`
	PropertyID            sql.NullString `db:"Property_ID"`
	PropertyLastName      sql.NullString `db:"Property_Last_Name"`
	PropertyLatitude      sql.NullString `db:"Property_Latitude"`
	PropertyLongitude     sql.NullString `db:"Property_Longitude"`
	PropertyState         sql.NullString `db:"Property_State"`
	PropertyStateAbbr     sql.NullString `db:"Property_State_Abbr"`
	PropertyStatus        sql.NullInt64  `db:"Property_Status"`
	PropertyStatusMessage sql.NullString `db:"Property_Status_Message"`
	PropertyZipCode       sql.NullString `db:"Property_ZipCode"`

	QualiaFlag         sql.NullInt64  `db:"Qualia_Flag"`
	RecipientAddress   sql.NullString `db:"Recipient_Address"`
	RecipientName      sql.NullString `db:"Recipient_Name"`
	RecipientPhone     sql.NullString `db:"Recipient_Phone"`
	ReportVersion      sql.NullString `db:"Report_Version"`
	SPDocInReport      sql.NullInt64  `db:"SP_DocInReport"`
	SPTemplate         sql.NullInt64  `db:"SP_Template"`
	SalesPrice         sql.NullString `db:"Sales_Price"`
	SearchDuration     sql.NullInt64  `db:"Search_Duration"`
	SettlementDate     sql.NullString `db:"Settlement_Date"`
	SourceTypeID       sql.NullInt64  `db:"Source_Type_ID"`
	SubscriptionStatus sql.NullString `db:"Subscription_Status"`
	TitlesID           sql.NullString `db:"Titles_ID"`

	UserFirstName           sql.NullString `db:"User_First_Name"`
	UserFullName            sql.NullString `db:"User_Full_Name"`
	UserLastName            sql.NullString `db:"User_Last_Name"`
	UserLoginName           sql.NullString `db:"User_Login_Name"`
	UserOrderExportToReport sql.NullInt64  `db:"User_Order_Export_To_Report"`
	UserRole                sql.NullInt64  `db:"User_Role"`
	UserStatus              sql.NullInt64  `db:"User_Status"`
	YearsRequired           sql.NullInt64  `db:"Years_Required"`

	// Derived/computed — populated after DB fetch or org query
	CompleteAddress  string
	ReportCoverInfo  sql.NullString
}

// ---------------------------------------------------------------------------
// Report_Versions row
// ---------------------------------------------------------------------------

type ReportVersion struct {
	ID          int            `db:"Id"`
	SpID        int            `db:"Sp_Id"`
	SPVersion   sql.NullString `db:"SP_Version"`
	JsonData    sql.NullString `db:"Report_JSON"`
	JsonUpdatAt sql.NullString `db:"Json_Update_At"`
	ModifiedAt  sql.NullString `db:"Modified_At"`
	CreatedAt   sql.NullString `db:"Created_At"`
	CreatedBy   sql.NullString `db:"Created_By"`
	ModifiedBy  sql.NullString `db:"Modified_By"`
}

// ---------------------------------------------------------------------------
// Builder data — mirrors getBuilderData() return shape
// ---------------------------------------------------------------------------

type BuilderData struct {
	SearchPackage                 SearchPackage
	SPParties                     []SPParty
	SPAssessments                 []map[string]any
	SPTaxes                       []map[string]any
	SPChainOfTitlesVD             []map[string]any
	SPChainOfTitlesCT             []map[string]any
	SPSecurityInstruments         []map[string]any
	SPLienJudgement               []map[string]any
	SPExceptionRestrictionAdverse []map[string]any
	GlobalCommitments             []map[string]any
	CommitmentTypings             []map[string]any
	SPDocuments                   []map[string]any
	SPGeneralComments             []map[string]any
}

// ---------------------------------------------------------------------------
// Search_Packages (Report_Builder table) row
// ---------------------------------------------------------------------------

type SearchPackage struct {
	ID                  int            `db:"Id"`
	OrderID             sql.NullInt64  `db:"Order_ID"`
	BuilderID           sql.NullString `db:"Builder_ID"`
	VDManualSort        sql.NullInt64  `db:"VD_Manual_Sort"`
	CTManualSort        sql.NullInt64  `db:"CT_Manual_Sort"`
	SIManualSort        sql.NullInt64  `db:"SI_Manual_Sort"`
	LJManualSort        sql.NullInt64  `db:"LJ_Manual_Sort"`
	ERManualSort        sql.NullInt64  `db:"ER_Manual_Sort"`
	PUD                 sql.NullInt64  `db:"PUD"`
	Parcel              sql.NullString `db:"Parcel"`
	Registry            sql.NullInt64  `db:"Registry"`
	InterestTypeID      sql.NullInt64  `db:"Interest_Type_Id"`
	IncludeDoesNotApply sql.NullInt64  `db:"Include_Does_Not_Apply"`
}

// ---------------------------------------------------------------------------
// SP_Parties / Alias
// ---------------------------------------------------------------------------

type SPParty struct {
	ID                int            `db:"Id"`
	SpID              int            `db:"Sp_Id"`
	EntityID          sql.NullInt64  `db:"Entity_ID"`
	FirstBusinessName sql.NullString `db:"First_Business_Name"`
	MiddleName        sql.NullString `db:"Middle_Name"`
	LastName          sql.NullString `db:"Last_Name"`
	Status            sql.NullInt64  `db:"Status"`
	Applies           sql.NullInt64  `db:"Applies"`
	Alias             []SPPartyAlias
}

type SPPartyAlias struct {
	ID      int            `db:"Id"`
	PartyID int            `db:"Party_ID"`
	Alias   sql.NullString `db:"Alias"`
	IsLegal sql.NullInt64  `db:"IsLegal"`
	Status  sql.NullInt64  `db:"Status"`
}

// ---------------------------------------------------------------------------
// Prep / lookup data — mirrors collectPrepData() + constructPayload()
// ---------------------------------------------------------------------------

type PrepData struct {
	DocTypes      []DocumentType
	TaxSources    []TaxType
	TaxStatus     []TaxPaidStatus
	ChainTypes    []ChainInstrumentType
	LienTypes     []LienJudgementType
	PartyEntities []PartyEntities

	SecInstEntities []SecurityInstrumentsType
	ERAEntities     []ERAEntities
	TaxEntities     []TaxEntities
	ChainEntities   []ChainOfTitleEntities
	LienEntities    []LienJudgementEntities
	TaxAuthorityType []TaxAuthorityType
	InterestTypes   []PropertyInterestType
}

// ---------------------------------------------------------------------------
// Top-level payload — mirrors the payload object built in createSearchPackage
// ---------------------------------------------------------------------------

type SearchPackagePayload struct {
	OrderData   *OrderPropertyData
	BuilderData *BuilderData
	VersionData *ReportVersion
	PrepData    *PrepData
}

// ---------------------------------------------------------------------------
// Lookup entity types (used by getAllMaps / PrepData)
// ---------------------------------------------------------------------------

type DocumentType struct {
	Id         int    `db:"Id"`
	Type       string `db:"Type"`
	TypeStatus bool   `db:"Type_Status"`
}

type TaxType struct {
	Id         int    `db:"Id"`
	SourceType string `db:"Source_Type"`
	TypeStatus bool   `db:"Source_Type_Status"`
}

type TaxPaidStatus struct {
	Id         int    `db:"Id"`
	StatusType string `db:"Status_Type"`
	Status     bool   `db:"Status"`
}

type RbSectionTypeMaps struct {
	DocTypesMap         map[int]DocumentType
	TaxSourcesMap       map[int]TaxType
	TaxPaidStatusMap    map[int]TaxPaidStatus
	ChainTypesMap       map[int]ChainInstrumentType
	LienTypesMap        map[int]LienJudgementType
	PartyEntitiesMap    map[int]PartyEntities
	SecInstEntitiesMap  map[int]SecurityInstrumentsType
	ERAEntitiesMap      map[int]ERAEntities
	TaxEntitiesMap      map[int]TaxEntities
	ChainEntitiesMap    map[int]ChainOfTitleEntities
	LienEntitiesMap     map[int]LienJudgementEntities
	TaxAuthorityTypeMap map[int]TaxAuthorityType
	InterestTypeMap     map[int]PropertyInterestType
}

type ChainInstrumentType struct {
	Id     int    `db:"Id"`
	Type   string `db:"Type"`
	TypeId int    `db:"Type_ID"`
	Status bool   `db:"Status"`
}

type LienJudgementType struct {
	Id     int    `db:"Id"`
	Type   string `db:"Type"`
	TypeId int    `db:"Type_ID"`
	Status bool   `db:"Status"`
}

type PartyEntities struct {
	Id           int    `db:"Id"`
	Entity       string `db:"Entity"`
	EntityStatus bool   `db:"Entity_Status"`
}

type SecurityInstrumentsType struct {
	Id           int    `db:"Id"`
	Entity       string `db:"Entity"`
	EntityStatus bool   `db:"Entity_Status"`
}

type ERAEntities struct {
	Id           int    `db:"Id"`
	Entity       string `db:"Entity"`
	EntityStatus bool   `db:"Entity_Status"`
}

type TaxEntities struct {
	Id           int    `db:"Id"`
	Entity       string `db:"Entity"`
	EntityStatus bool   `db:"Entity_Status"`
}

type ChainOfTitleEntities struct {
	Id           int    `db:"Id"`
	Entity       string `db:"Entity"`
	EntityStatus bool   `db:"Entity_Status"`
}

type LienJudgementEntities struct {
	Id           int    `db:"Id"`
	Entity       string `db:"Entity"`
	EntityStatus bool   `db:"Entity_Status"`
}

type TaxAuthorityType struct {
	Id   int    `db:"Id"`
	Name string `db:"Name"`
}

type PropertyInterestType struct {
	Id               int    `db:"Id"`
	InterestTypeName string `db:"Interest_Type_Name"`
}

// ---------------------------------------------------------------------------
// Optimized / curated report — output of optimizeReportData()
// Mirrors the finalizedData object assembled by finalizeData() in package_helper.js
// ---------------------------------------------------------------------------

// OptimizedReport is the fully cleansed, display-ready report structure.
type OptimizedReport struct {
	PropertyDetails     map[string]any   `json:"Property_Details"`
	ReportDetails       map[string]any   `json:"ReportDetails"` // same reference as PropertyDetails (JS does this too)
	SearchParties       []map[string]any `json:"Search_Parties"`
	Assessments         []map[string]any `json:"Assessments"`
	Taxes               []map[string]any `json:"Taxes"`
	VestingDeed         []map[string]any `json:"Vesting_Deed"`
	ChainOfTitle        []map[string]any `json:"Chain_Of_Title"`
	SecurityInstruments []map[string]any `json:"Security_Instruments"`
	LienAndJudgement    []map[string]any `json:"LienAndJudgement"`
	ERA                 []map[string]any `json:"ERA"`
	GeneralComments     []map[string]any `json:"General_Comments"`
	ReportChanges       []string         `json:"reportChanges"`
	SPVersion           any              `json:"SP_Version"`
}

// ReportPayload is the full data bag passed into generateHTMLTemplate.
// It mirrors the combination of `payload` + template-helper fields set by
// generateHTMLTemplate in pdf_helper.js.
type ReportPayload struct {
	Report      OptimizedReport    `json:"Report"`
	OrderData   *OrderPropertyData `json:"OrderData"`
	BuilderData *BuilderData       `json:"BuilderData"`
	PrepData    *PrepData          `json:"PrepData"`
	VersionData *ReportVersion     `json:"VersionData"`

	// Template helpers (populated by generateHTMLTemplate)
	Owners             []string            `json:"owners"`
	NamesSearched      []NamesSearchedItem `json:"namesSearched"`
	GoogleMapImage     string              `json:"googleMapImage"`
	GoogleMapImageSecond string            `json:"googleMapImageSecond"`
	SearchDate         string              `json:"searchDate"`
	SearchTime         string              `json:"searchTime"`
	EstOrEdt           string              `json:"estOrEdt"`
	EffectiveDate      string              `json:"effectiveDate"`
	EffectiveTime      string              `json:"effectiveTime"`
	SearchStartDate    string              `json:"searchStartDate"`
	PriorEffectiveDate string              `json:"priorEffectiveDate"`
	YrsSearch          any                 `json:"yrsSearch"`
	FooterTxt          string              `json:"footerTxt"`
	Disclaimer         string              `json:"Disclaimer"`
	AssetDir           string              `json:"AssetDir"`
	IncludeDoesNotApply int                `json:"IncludeDoesNotApply"`

	// Asset paths (needed by templates)
	LogoPath       string `json:"logopath"`
	PippinReport   string `json:"pippinreport"`
	CustLogo       string `json:"custlogo"`
	BuildingPath   string `json:"buildingpath"`
	FooterPath     string `json:"footerpath"`
	FooterPathCust string `json:"footerpathcust"`
	SealPath       string `json:"sealpath"`
	MenLogo        string `json:"menLogo"`
	EffectiveLogo  string `json:"effectiveLogo"`
}

// NamesSearchedItem mirrors the objects pushed into namesSearchedList in pdf_helper.js.
type NamesSearchedItem struct {
	Name      string
	JudgLabel string
	LienLabel string
	Judgments int
	Liens     int
}

// SortedDoc mirrors the { path, idx } object in getSortedDocList() (package_helper.js).
type SortedDoc struct {
	Path map[string]any
	Idx  int
}
