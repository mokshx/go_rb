package rb

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
)

type App struct {
	DB *sql.DB
}

type RbRequest struct {
	OrderId string
}

type ReportDetails struct {
	SearchPackage                    SearchPackage
	SP_Assesments                    SP_Assesments
	SP_Taxes                         SP_Taxes
	SP_Chain_Of_Titles_CT            SP_Chain_Of_Titles_CT
	SP_Chain_Of_Titles_VD            SP_Chain_Of_Titles_VD
	SP_Security_Instruments          SP_Security_Instruments
	SP_Lien_Judgement                SP_Lien_Judgement
	SP_Exception_Restriction_Adverse SP_Exception_Restriction_Adverse
	Global_Commitments               Global_Commitments
	Commitment_Typings               Commitment_Typings
	SP_Documents                     SP_Documents
	SP_General_Comments              SP_General_Comments
}

type SearchPackage struct{}

type SP_Parties struct{}

type SP_Assesments struct{}

type SP_Taxes struct{}

type SP_Chain_Of_Titles_VD struct{}

type SP_Chain_Of_Titles_CT struct{}

type SP_Security_Instruments struct{}

type SP_Lien_Judgement struct{}

type SP_Exception_Restriction_Adverse struct{}

type Global_Commitments struct{}

type Commitment_Typings struct{}

type SP_Documents struct{}

type SP_General_Comments struct{}

func (a *App) CreateSearchPackage(req RbRequest) {
	orderData := getOrderPropertyData(a.DB, req.OrderId)

	maps := getAllMaps(a.DB)
	log.Print(maps)
	fmt.Println(orderData.AbstractorUserFirstName)
}

func getAllMaps(db *sql.DB) RbSectionTypeMaps {
	ma := getDocTypeIdMap(db)
	taxSources := getTaxSourceTypeIdMap(db)
	taxPaidStatusMap := getTaxPaidStatusMap(db)
	chainTypeMap := getChainTypesIdMap(db)
	ljMap := getLienTypesIdMap(db)
	sp := getSPPartyIdMap(db)
	secInst := getSecInstrumentsIdMap(db)
	era := getEraIdMap(db)
	taxEntities := getTaxEntitiesIdMap(db)
	chainEntities := getChainOfTitleTypeIdMap(db)
	lienEntities := getLienJudgementIdMap(db)
	taxAuth := getTaxAuthorityTypeMap(db)
	interestTypes := getPropertyInterestTypes(db)

	return RbSectionTypeMaps{
		docTypesMap:         ma,
		taxSourcesMap:       taxSources,
		taxPaidStatusMap:    taxPaidStatusMap,
		chainTypesMap:       chainTypeMap,
		lienTypesMap:        ljMap,
		partyEntitiesMap:    sp,
		secInstEntitiesMap:  secInst,
		eraEntitiesMap:      era,
		taxEntitiesMap:      taxEntities,
		chainEntitiesMap:    chainEntities,
		lienEntitesMap:      lienEntities,
		taxAuthorityTypeMap: taxAuth,
		interestTypeMap:     interestTypes,
	}
}

func getOrderPropertyData(db *sql.DB, orderId string) OrderPropertyData {
	query := `
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
			User_Login_Name
		FROM Order_Property_User_View
		WHERE Order_ID = ?
	`

	var data OrderPropertyData

	err := db.QueryRow(query, orderId).Scan(
		&data.PropertyID,
		&data.OrderID,
		&data.CustomerID,
		&data.FileID,
		&data.PropertyAddress1,
		&data.PropertyAddress2,
		&data.PropertyCity,
		&data.PropertyCounty,
		&data.PropertyState,
		&data.PropertyStateAbbr,
		&data.PropertyZipCode,
		&data.PropertyLatitude,
		&data.PropertyLongitude,
		&data.OrderCreationDate,
		&data.OrderModificationDate,
		&data.OrderStatus,
		&data.OrderProductStatus,
		&data.ProductID,
		&data.ProductDescription,
		&data.OrganizationID,
		&data.OrganizationName,
		&data.UserFirstName,
		&data.UserLastName,
		&data.UserFullName,
		&data.UserLoginName,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return OrderPropertyData{}
		}

		log.Printf("failed to get order property data for orderId %s: %v", orderId, err)
		return OrderPropertyData{}
	}

	return data
}

func printSingleRow(db *sql.DB, query string, args ...any) {
	rows, err := db.Query(query, args...)
	if err != nil {
		log.Print("Error in querying the values ", err)
		return
	}

	if !rows.Next() { // cause single row
		log.Print("No rows found")
		return
	}

	defer rows.Close()
	columns, err := rows.Columns()

	if err != nil {
		log.Print("Error in fetching columns", err)
		return
	}

	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))

	for i, _ := range columns {
		valuePtrs[i] = &values[i]
	}

	err = rows.Scan(valuePtrs...)
	if err != nil {
		log.Print("error in scanning ", err)
		return
	}

	rowMap := map[string]any{}

	for i, col := range columns {
		value := values[i]

		if bytes, ok := value.([]byte); ok {
			rowMap[col] = string(bytes)
		} else {
			rowMap[col] = value
		}
	}

	jsonBytes, err := json.MarshalIndent(rowMap, "", " ")
	fmt.Println(string(jsonBytes))
}

func getDocTypeIdMap(db *sql.DB) map[int]DocumentType {
	docTypeMap := make(map[int]DocumentType)
	query := "Select Type, Id, Type_Status from SP_Document_Types"
	rows, err := db.Query(query)
	if err != nil {
		log.Print("Error in getDocTypeMap query ", err)
	}
	defer rows.Close()
	for rows.Next() {
		var d DocumentType
		err := rows.Scan(&d.Type, &d.Id, &d.TypeStatus)
		if err != nil {
			log.Print("Error in getDocTypeMap Scanning ", err)
		}
		if d.TypeStatus {
			// only add to map if the type status is valid
			docTypeMap[d.Id] = d
		}
	}
	return docTypeMap
}

func getTaxSourceTypeIdMap(db *sql.DB) map[int]TaxType {
	taxTypeMap := make(map[int]TaxType)
	query := "Select Id, Source_Type, Source_Type_Status from SP_Tax_Source_Types"

	rows, err := db.Query(query)
	if err != nil {
		log.Print("Error n query of the tax typeId", err)
	}
	defer rows.Close()

	for rows.Next() {
		var d TaxType
		err = rows.Scan(&d.Id, &d.SourceType, &d.TypeStatus)
		if err == nil && d.TypeStatus {
			taxTypeMap[d.Id] = d
		}
	}
	return taxTypeMap
}

func getTaxPaidStatusMap(db *sql.DB) map[int]TaxPaidStatus {
	taxPaidStatusMap := make(map[int]TaxPaidStatus)

	rows, err := db.Query("Select Id, Status, Status_Type from SP_Tax_Paid_Status_Types")

	if err != nil {
		log.Print("Error in query tax paid status ", err)
		return taxPaidStatusMap
	}
	defer rows.Close()
	for rows.Next() {
		var t TaxPaidStatus
		err = rows.Scan(&t.Id, &t.Status, &t.StatusType)
		if err != nil {
			log.Print("Error in scanning ", err)
		}
		if t.Status {
			taxPaidStatusMap[t.Id] = t
		}
	}

	return taxPaidStatusMap

}

func getChainTypesIdMap(db *sql.DB) map[int]ChainInstrumentType {
	chainInstrumentMap := make(map[int]ChainInstrumentType)
	rows, err := db.Query("Select Id, Type, Type_ID, Status from SP_Chain_Of_Title_Instrument_Types")

	if err != nil {
		log.Print("Error in query of getChainTypesIdMap ", err)
		return chainInstrumentMap
	}

	defer rows.Close()

	for rows.Next() {
		var d ChainInstrumentType
		err = rows.Scan(&d.Id, &d.Type, &d.TypeId, &d.Status)
		if err != nil {
			log.Print("Error in scaninng ChainInstrumentType, ", err)
			return chainInstrumentMap
		}

		if d.Status {
			chainInstrumentMap[d.Id] = d
		}
	}
	return chainInstrumentMap
}

func getLienTypesIdMap(db *sql.DB) map[int]LienJudgementType {
	ljMap := make(map[int]LienJudgementType)

	rows, err := db.Query(`SELECT Id, Type, Type_ID, Status  from SP_Lien_Judgement_Types`)

	if err != nil {
		log.Print("Error in get lien type id map ", err)
		return ljMap
	}

	defer rows.Close()

	for rows.Next() {
		var lj LienJudgementType
		err = rows.Scan(&lj.Id, &lj.Type, &lj.TypeId, &lj.Status)
		if err != nil {
			log.Print("Error in LInet  judmgent scan ", err)
			return ljMap
		}
		if lj.Status {
			ljMap[lj.Id] = lj
		}
	}

	return ljMap
}

func getSPPartyIdMap(db *sql.DB) map[int]PartyEntities {
	spMap := make(map[int]PartyEntities)
	rows, err := db.Query("select Id, Entity, Entity_Status from SP_Party_Entities")
	if err != nil {
		log.Print("Eerr in Partiy entities ", err)
		return spMap
	}
	defer rows.Close()

	for rows.Next() {
		var p PartyEntities
		err = rows.Scan(&p.Id, &p.Entity, &p.EntityStatus)
		if err != nil {
			log.Print("Error in scanning party entity ", err)
			return spMap
		}
		if p.EntityStatus {
			spMap[p.Id] = p
		}
	}
	return spMap
}

func getSecInstrumentsIdMap(db *sql.DB) map[int]SecurityInstrumentsType {
	smap := make(map[int]SecurityInstrumentsType)
	rows, err := db.Query("Select Id, Entity, Entity_Status from SP_Security_Instruments_Entities")

	if err != nil {
		log.Print("Error in query getSecInstrumentsIdMap ", err)
		return smap
	}

	defer rows.Close()

	for rows.Next() {
		var d SecurityInstrumentsType
		err = rows.Scan(&d.Id, &d.Entity, &d.EntityStatus)
		if err != nil {
			log.Print("Erro rin scannign ", err)

		}
		if d.EntityStatus {
			smap[d.Id] = d
		}
	}

	return smap
}

func getEraIdMap(db *sql.DB) map[int]ERAEntities {
	eraMap := make(map[int]ERAEntities)
	rows, err := db.Query("select Id, Entity, Entity_Status from SP_Exception_Restriction_Adverse_Entities")
	if err != nil {
		log.Print("Error in ERA entities query ", err)
		return eraMap
	}
	defer rows.Close()

	for rows.Next() {
		var e ERAEntities
		err = rows.Scan(&e.Id, &e.Entity, &e.EntityStatus)
		if err != nil {
			log.Print("Error in scanning ERA entity ", err)
			return eraMap
		}
		if e.EntityStatus {
			eraMap[e.Id] = e
		}
	}
	return eraMap
}

func getTaxEntitiesIdMap(db *sql.DB) map[int]TaxEntities {
	taxMap := make(map[int]TaxEntities)
	rows, err := db.Query("select Id, Entity, Entity_Status from SP_Tax_Entities")
	if err != nil {
		log.Print("Error in Tax entities query ", err)
		return taxMap
	}
	defer rows.Close()

	for rows.Next() {
		var t TaxEntities
		err = rows.Scan(&t.Id, &t.Entity, &t.EntityStatus)
		if err != nil {
			log.Print("Error in scanning Tax entity ", err)
			return taxMap
		}
		if t.EntityStatus {
			taxMap[t.Id] = t
		}
	}
	return taxMap
}

func getChainOfTitleTypeIdMap(db *sql.DB) map[int]ChainOfTitleEntities {
	chainMap := make(map[int]ChainOfTitleEntities)
	rows, err := db.Query("select Id, Entity, Entity_Status from SP_Chain_Of_Title_Entities")
	if err != nil {
		log.Print("Error in Chain of Title entities query ", err)
		return chainMap
	}
	defer rows.Close()

	for rows.Next() {
		var c ChainOfTitleEntities
		err = rows.Scan(&c.Id, &c.Entity, &c.EntityStatus)
		if err != nil {
			log.Print("Error in scanning Chain of Title entity ", err)
			return chainMap
		}
		if c.EntityStatus {
			chainMap[c.Id] = c
		}
	}
	return chainMap
}

func getLienJudgementIdMap(db *sql.DB) map[int]LienJudgementEntities {
	lienMap := make(map[int]LienJudgementEntities)
	rows, err := db.Query("select Id, Entity, Entity_Status from SP_Lien_Judgement_Entities")
	if err != nil {
		log.Print("Error in Lien Judgement entities query ", err)
		return lienMap
	}
	defer rows.Close()

	for rows.Next() {
		var l LienJudgementEntities
		err = rows.Scan(&l.Id, &l.Entity, &l.EntityStatus)
		if err != nil {
			log.Print("Error in scanning Lien Judgement entity ", err)
			return lienMap
		}
		if l.EntityStatus {
			lienMap[l.Id] = l
		}
	}
	return lienMap
}

func getTaxAuthorityTypeMap(db *sql.DB) map[int]TaxAuthorityType {
	taxAuthMap := make(map[int]TaxAuthorityType)
	rows, err := db.Query("select Id, Name from SP_Tax_Authority_Types")
	if err != nil {
		log.Print("Error in Tax Authority Types query ", err)
		return taxAuthMap
	}
	defer rows.Close()

	for rows.Next() {
		var t TaxAuthorityType
		err = rows.Scan(&t.Id, &t.Name)
		if err != nil {
			log.Print("Error in scanning Tax Authority Type ", err)
			return taxAuthMap
		}
		taxAuthMap[t.Id] = t
	}
	return taxAuthMap
}

func getPropertyInterestTypes(db *sql.DB) map[int]PropertyInterestType {
	propIntMap := make(map[int]PropertyInterestType)
	rows, err := db.Query("select Id, Interest_Type_Name from Property_Interest_Types")
	if err != nil {
		log.Print("Error in Property Interest Types query ", err)
		return propIntMap
	}
	defer rows.Close()

	for rows.Next() {
		var p PropertyInterestType
		err = rows.Scan(&p.Id, &p.InterestTypeName)
		if err != nil {
			log.Print("Error in scanning Property Interest Type ", err)
			return propIntMap
		}
		propIntMap[p.Id] = p
	}
	return propIntMap
}
