package storage

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"

	// "strings"
	_ "github.com/mattn/go-sqlite3"
)

type Transaction struct{
	Data map[string]string
}
type MergedTransaction struct{
	Data string
	Index int
}

// OpenDatabase opens and returns a connection to the SQLite database at the given path
func OpenDatabase(DBpath string) *sql.DB { 
	db, err := sql.Open("sqlite3", DBpath)
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func DisplayColumns(db *sql.DB, tableName string, columns []string, limit int) {

	cols := ""
	for i, v := range columns {
		if i != len(columns) - 1 {
			cols += v + ", "
		} else {
			cols += v
		}
	}
	
	query := fmt.Sprintf("SELECT %s FROM %s LIMIT %d", cols, tableName, limit)
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	columnNames, err := rows.Columns()
	if err != nil {
		log.Fatal(err)
	}
	
	values := make([]interface{}, len(columnNames))
	valuePtrs := make([]interface{}, len(columnNames))
	for i := range columnNames {
		valuePtrs[i] = &values[i]
	}
	
	for rows.Next() {
		err = rows.Scan(valuePtrs...)
		if err != nil {
			log.Fatal(err)
		}
		
		for i, col := range columnNames {
			val := values[i]
			switch v := val.(type) {
			case []byte:
				fmt.Printf("%s: %s ", col, string(v))
			default:
				fmt.Printf("%s: %v ", col, v)
			}
		}
		fmt.Println()
	}
	
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
}

func RetriveData(db *sql.DB,tableName string,columns,mergedColumns []string,limit int) []Transaction {
	existingColumns := GetTableColumns(db, tableName)
	validColumns := make([]string, 0)
	
	for _, col := range columns {
		found := false
		for _, existingCol := range existingColumns {
			if strings.EqualFold(col, existingCol) {
				// Use the exact case from the database to avoid case sensitivity issues
				validColumns = append(validColumns, existingCol)
				found = true
				break
			}
		}
		
		if !found {
			fmt.Printf("Warning: Column '%s' not found in table '%s'. Skipping.\n", col, tableName)
		}
	}
	
	if len(validColumns) == 0 {
		fmt.Printf("Error: No valid columns found in table '%s'.\n", tableName)
		return nil
	}
	cols := ""
	for i, v := range validColumns {
		if i != len(columns) - 1 {
			cols += v + ", "
		} else {
			cols += v
		}
	}
	Query := fmt.Sprintf("select %s from %s limit %s",cols,tableName,fmt.Sprint(limit))
	rows,err := db.Query(Query)

	if err != nil{
		log.Fatal(err)
	}
	defer rows.Close()

	columnsName,err := rows.Columns()
	if err != nil{
		log.Fatal(err)
	}
	values := make([]interface{},len(columnsName))
	valuePtr := make([]interface{},len(columnsName))
	for i := range columnsName{
		valuePtr[i] = &values[i]
	}
	var records []Transaction
	for rows.Next(){
		err := rows.Scan(valuePtr...)
		if err != nil{
			log.Fatal(err)
		}
		trans := Transaction{
			Data:make(map[string]string),
		}

		for i,col := range columnsName{
			var valStr string
			val := values[i]
			switch v := val.(type){
			case []byte:
				valStr = string(v)
			default:
				valStr = fmt.Sprintf("%v",v)
			}
			trans.Data[col] = valStr

		}
		records = append(records, trans)
	}
	return records
}

func GetTableColumns(db *sql.DB, tableName string) []string {
	query := fmt.Sprintf("PRAGMA table_info(%s)",tableName)
	rows,err := db.Query(query)

	if err != nil{
		log.Fatal(err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next(){
		var cid, notnull,pk int
		var name, typename string
		var defaultValue interface{}
		if err := rows.Scan(&cid,&name,&typename,&notnull,&defaultValue,&pk); err != nil{
			log.Fatal(err)
		}
		columns = append(columns, name)
	}

	return columns

}


func sanitizedColumnName(name string) string {
	reg := regexp.MustCompile("[^a-zA-Z0-9_]")
	sanitized := reg.ReplaceAllString(name,"_")
	if len(sanitized) > 0 && sanitized[0] >= '0' && sanitized[0] <= '9'{
		sanitized = "-" + sanitized
	}
	return sanitized
}


func CreateDatabase(transactions []Transaction,tableName string,columns []string,dbPath string){
	db,err := sql.Open("sqlite3",dbPath)
	if err != nil{
		fmt.Printf("Error creating Database%v\n",err)
		return
	}

	defer db.Close()

	sanitizedColumns := make([]string,len(columns))
	columnMap := make(map[string]string)
	for i,col := range columns{
		sanitized := sanitizedColumnName(col)
		sanitizedColumns[i] = sanitized
		columnMap[col] = sanitized

		if sanitized != col{
			fmt.Printf("Note: column name %s is sanitized to %s for sql compatibility\n",col,sanitized)
		}
	}

	columnSql := strings.Join(sanitizedColumns," TEXT, ") + " TEXT"
	createSqlTable := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s);",tableName,columnSql)
	fmt.Println("Creating a table...",createSqlTable)
	
	_, err = db.Exec(createSqlTable)
	if err != nil{
		fmt.Printf("Error Creating table %v\n",err)
		return
	}
	existingColumns := GetTableColumns(db,tableName)

	if len(existingColumns) == 0{
		fmt.Printf("Couldn't Verify table structure after creation. \n")
		return
	}

	placeholders := strings.Repeat("?,", len(sanitizedColumns))
	placeholders = strings.TrimRight(placeholders, ",")
	insertSql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",tableName,strings.Join(sanitizedColumns,", "),placeholders)
	fmt.Println("inserting with sql....",insertSql)
	stmt,err := db.Prepare(insertSql)

	if err != nil{
		fmt.Printf("preparing insert statement: %v\n",err)
		return
	}
	defer stmt.Close()

	successCount := 0
	errorCount := 0

	for _,tran := range transactions{
		values := make([]interface{},len(sanitizedColumns))

		for i , originalCol := range columns{
			values[i] = tran.Data[originalCol]
		}

		_,err := stmt.Exec(values...)

		if err != nil{
			errorCount++
			fmt.Printf("Error inserting statement: %v\n",err)
			
			if errorCount > 10{
				continue
			}
		} else {
			successCount++
		}

	}	
	if errorCount > 0 {
		fmt.Printf("\n Insert Summery: %d successful, %d failed",successCount,errorCount)
	} else {
		fmt.Printf("All %d rows are successfully inserted \n",successCount)
	}
}


