package storage

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/SanthoshCheemala/FLARE/pkg/matrix"
)

func SaveSecretkey(sk *matrix.Vector,dbPath string) error{
	dir := strings.Split(dbPath,"/")

	if len(dir) > 1{
		dirPath := strings.Join(dir[:len(dir)-1],"/")
		if _,err := os.Stat(dirPath); os.IsNotExist(err) {
			if err := os.MkdirAll(dirPath,0755); err != nil{
				return err
			}
		}
	}

	file ,err := os.Create(dbPath)
	if err != nil{
		return err
	}
	defer file.Close()

	skBytes := sk.Encode()
	
	for _,bytes := range skBytes{
		lenBytes := []byte{byte(len(bytes))}
		if _,err := file.Write(lenBytes); err != nil{
			return err
		}
		if _,err := file.Write(bytes); err != nil{
			return err
		}
	}
	return nil
}

func InitializeTreeDB(db *sql.DB, layers int) error {
    for i := 0; i <= layers; i++ {
        query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS tree_%d (p1 BLOB, p2 BLOB, P3 BLOB, p4 BLOB, y_def BOOLEAN)", i)
        _, err := db.Exec(query)
        if err != nil {
            return fmt.Errorf("error creating tree table %d: %w", i, err)
        }
    }
    return nil
}