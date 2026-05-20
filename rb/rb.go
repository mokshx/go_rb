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
	log.Print(orderData)
	fmt.Println(orderData)
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
