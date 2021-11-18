package modules

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"

	"github.com/extrame/xls"
	"github.com/tealeg/xlsx"
)

func ReadCSV(csvFile string) ([][]string, error) {
	f, err := os.Open(csvFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1 // set return record number per row, -1 means `all` record

	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
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
func ReadXLSX(xlsxFile string) error {
	x, err := xlsx.OpenFile(xlsxFile)
	if err != nil {
		return err
	}
	log.Printf("parsing XLS file %s, with sheets Num. %d", xlsxFile, len(x.Sheets))
	res := [][]string{}

	for index, sheet := range x.Sheets {
		if index == 0 {
			fmt.Println(sheet.Name)
			temp := make([][]string, len(sheet.Rows))
			for k, row := range sheet.Rows {
				data := []string{}
				for _, cell := range row.Cells {
					data = append(data, cell.Value)
				}
				temp[k] = data
			}

			res = append(res, temp...)
		}
	}
	fmt.Println(res)
	return nil
}

func ReadXLS(xlsFile, charset string) error {

	// var res = [][]string{}
	wb, err := xls.Open(xlsFile, charset)
	if err != nil {
		return err
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
				for r := 0; r < (int(sheet.MaxRow)); r++ {
					row := sheet.Row(r)
					// data := make([]string, 0)
					data := []string{}

					if row.LastCol() > 0 {
						for c := 0; c < row.LastCol(); c++ {
							col := row.Col(c)
							data = append(data, col)
							// write to db here
						}
						// sheet1[r] = data
						fmt.Printf("read line:%d data %s\n", r, data)
					}

				}
				// res = append(res, sheet1...)
			}

		}
	}

	return nil
}
