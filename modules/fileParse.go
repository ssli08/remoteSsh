package modules

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"sshtunnel/database"
	"strings"

	"github.com/extrame/xls"
	"github.com/tealeg/xlsx"
)

// var sql = "INSERT INTO instances (instance_name,public_ip,private_ip,region,project) values ('%s','%s','%s','%s','%s')"

func ReadCSV(db *sql.DB, csvFile string) ([][]string, error) {
	f, err := os.Open(csvFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s with error %s only support CSV suffix file", csvFile, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1 // set return record number per row, -1 means `all` record

	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	for _, data := range records {
		if strings.Contains(data[1], "instance_name") {
			continue
		}
		sql := fmt.Sprintf(`INSERT INTO %s 
				(instance_name,public_ip,private_ip,region,project,role,instance_type,instance_id) 
				values 
				('%s','%s','%s','%s','%s','%s','%s','%s')`, database.InstanceTableName, data[1], data[2], data[3], data[4], data[6], data[7], data[10], data[11])
		if err := database.DBExecute(db, sql); err != nil {
			log.Fatal(err)
		}
	}
	return records, nil
}
func WriteCSV(csvFile string, records [][]string) error {
	// example records
	/*
		records := [][]string{
			{"first_name", "last_name", "username"},
			{"Rob", "Pike", "rob"},
			{"Ken", "Thompson", "ken"},
			{"Robert", "Griesemer", "gri"},
		}
	*/

	f, err := os.OpenFile(csvFile, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	/*
		for _, record := range records {
			if err := w.Write(record); err != nil {
				return fmt.Errorf("error writing record (%s) to %s", record, csvFile)
			}
		}
		w.Flush()
	*/
	if err := w.WriteAll(records); err != nil {
		return err
	}
	if err := w.Error(); err != nil {
		return err
	}

	return nil
}
func ReadXLSX(db *sql.DB, xlsxFile string) error {
	x, err := xlsx.OpenFile(xlsxFile)
	if err != nil {
		return err
	}

	log.Printf("parsing XLSX file %s, with sheets Num. %d", xlsxFile, len(x.Sheets))
	res := [][]string{}

	for index, sheet := range x.Sheets {
		if index == 0 {
			fmt.Println("sheet name: ", sheet.Name)
			temp := make([][]string, len(sheet.Rows))
			for k, row := range sheet.Rows {
				data := []string{}
				for _, cell := range row.Cells {
					data = append(data, cell.Value)
				}
				temp[k] = data
				if strings.Contains(data[1], "instance_name") {
					continue
				}
				// sql := fmt.Sprintf(`INSERT INTO %s
				// (instance_name,public_ip,private_ip,region,project)
				// values
				// ('%s','%s','%s','%s','%s')`, database.InstanceTableName, data[0], data[1], data[2], data[3], data[4])

				sql := fmt.Sprintf(`INSERT INTO %s 
				(id,instance_name,public_ip,private_ip,region,project,insert_time,role) 
				values 
				('%s','%s','%s','%s','%s','%s','%s',"%s")`, database.InstanceTableName, data[0], data[1], data[2], data[3], data[4], data[5], data[6], "data[7]")
				if err := database.DBExecute(db, sql); err != nil {
					log.Fatal(err)
				}
			}

			res = append(res, temp...)

		}
	}
	log.Println(res)
	return nil
}

func ReadXLS(db *sql.DB, xlsFile, charset string) error {
	// var res = [][]string{}
	wb, err := xls.Open(xlsFile, charset)
	if err != nil {
		return fmt.Errorf("failed to read %s with error %s only xls suffix file can be processed", xlsFile, err)
	}
	log.Printf("parsing XLS file %s, author: %s", xlsFile, wb.Author)

	numsh := wb.NumSheets()
	log.Printf("total %d worksheets need to be parsed\n", numsh)

	for n := 0; n <= numsh; n++ {
		if sheet := wb.GetSheet(n); sheet != nil {
			fmt.Println()
			log.Printf("Total Rows %d need to read in WorkSheet %s\n", sheet.MaxRow, sheet.Name)
			if sheet.MaxRow != 0 {
				// sheet1 := make([][]string, sheet.MaxRow)
				// sheet.MaxRow less than the really rows in worksheet, so plus 1 (int(sheet_MaxRow) + 1)
				for r := 0; r < (int(sheet.MaxRow) + 1); r++ {
					row := sheet.Row(r)
					// data := make([]string, 0)
					data := []string{}

					if row.LastCol() > 0 {
						for c := 0; c < row.LastCol(); c++ {
							col := row.Col(c)
							data = append(data, col)
						}
						// sheet1[r] = data
						// fmt.Printf("read line:%d data %s\n", r, data)
						if strings.Contains(data[1], "instance_name") {
							continue
						}
						// sql := fmt.Sprintf(`INSERT INTO %s
						// (instance_name,public_ip,private_ip,region,project)
						// values
						// ('%s','%s','%s','%s','%s')`, database.InstanceTableName, data[0], data[1], data[2], data[3], data[4])
						sql := fmt.Sprintf(`INSERT INTO %s (
							id,instance_name,public_ip,private_ip,region,project,insert_time,role) 
							values 
							('%s','%s','%s','%s','%s','%s','%s','%s')`, database.InstanceTableName, data[0], data[1], data[2], data[3], data[4], data[5], data[6], "data[7]")
						if err := database.DBExecute(db, sql); err != nil {
							log.Fatal(err)
						}
					}

				}
				// res = append(res, sheet1...)
			}

		}
	}

	return nil
}
