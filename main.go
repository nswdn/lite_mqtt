package main

import (
	"fmt"
	"github.com/tealeg/xlsx"
)

func main() {
	writeFile()
}

func read(filePath string) {
	file, err := xlsx.OpenFile(filePath)
	if err != nil {
		panic(err)
	}

	for _, sheet := range file.Sheets {
		_ = sheet.Name
		for _, row := range sheet.Rows {
			var strs []string
			for _, cell := range row.Cells {
				text := cell.String()
				strs = append(strs, text)
			}
		}
	}
}

func writeFile() {
	var file *xlsx.File
	var sheet *xlsx.Sheet
	var row *xlsx.Row
	var cell *xlsx.Cell
	var err error

	file = xlsx.NewFile()
	sheet, err = file.AddSheet("Sheet1")
	if err != nil {
		fmt.Printf(err.Error())
	}
	row = sheet.AddRow()
	row.SetHeightCM(0.5)
	cell = row.AddCell()
	cell.Value = "单位"
	cell = row.AddCell()
	cell.Value = "业务系统"
	cell = row.AddCell()
	cell.Value = "进程名"
	cell = row.AddCell()
	cell.Value = "V1000"
	cell = row.AddCell()
	cell.Value = "V2000"
	cell = row.AddCell()
	cell.Value = "H"
	cell = row.AddCell()
	cell.Value = "L"
	cell = row.AddCell()
	cell.Value = "A"
	cell = row.AddCell()
	cell.Value = "TIME"

	file.Save("test.xlsx")
}
