package main

import (
	"fmt"
	"github.com/xuri/excelize/v2"
	"strconv"
	"strings"
)

// Read responses from XLSX file exported from Microsoft Forms.
// Expect row 1 to have column titles in it with rows 2+ having the data (excel table format)
func readXLSX(filename string) ([]Response, error) {
	var (
		startcol, colincrement int
	)
	f, err := excelize.OpenFile(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rows, err := f.GetRows("Sheet1", excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, err
	}
	list := make([]Response, 0)
	for rownum, row := range rows {
		var a Response
		if rownum == 0 {
			// First row (row #1), so check titles to see if this spreadsheet is of the expected format
			if Questions, startcol, colincrement, err = checkXLSTitles(row); err != nil {
				return nil, err
			}
		} else {
			a.Email = strings.ToLower(row[3])
			a.Name = row[4]
			if TeamMode {
				if a.Team, err = findTeam(a.Email); err != nil {
					return nil, err
				}
			}
			f, err := strconv.ParseFloat(row[2], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid Completed time (%s) on row:\n%+v\n", err, row)
			}
			a.Completed = TimeFromExcelTime(f, false)
			for i := 0; i < len(Questions); i++ {
				colIdx := i*colincrement + startcol
				if colIdx < len(row) {
					a.Answers = append(a.Answers, strings.TrimSpace(row[colIdx]))
				}
			}
			a.AnswerScore = make([]int, len(Questions))
			list = append(list, a)
		}
	}
	return list, nil
}

func checkXLSTitles(row []string) ([]Question, int, int, error) {
	var cols = []string{"ID", "Start time", "Completion time", "Email", "Name"}
	if len(row) < 7 {
		return nil, 0, 0, fmt.Errorf("spreadsheet cannot be correct -- has less than 7 columns")
	}
	questions := make([]Question, 0)
	for i, c := range cols {
		if row[i] != c {
			return nil, 0, 0, fmt.Errorf("column #%d is not %s", i+1, c)
		}
	}
	startcol, colincrement := 7, 3
	if row[5] != "Total points" {
		startcol, colincrement = 5, 1
	}
	for i := startcol; i < len(row); i += colincrement {
		text := row[i]
		if len(text) == 0 {
			return nil, 0, 0, fmt.Errorf("column #%d has no title", i)
		}
		if colincrement > 1 {
			if i+2 >= len(row) {
				return nil, 0, 0, fmt.Errorf("too few columns, expect: answer, points, feedback tuples")
			}
			x := "Points - " + text
			if row[i+1] != x && strings.HasPrefix(row[i+1], "Column") == false {
				return nil, 0, 0, fmt.Errorf("column #%d was supposed to be '%s' but is '%s'", i+1+1, x, row[i+1])
			}
			x = "Feedback - " + text
			if row[i+2] != x && strings.HasPrefix(row[i+2], "Column") == false {
				return nil, 0, 0, fmt.Errorf("column #%d was supposed to be '%s' but is '%s'", i+2+1, x, row[i+2])
			}
		}
		if q, err := buildQuestion(row[i]); err != nil {
			return nil, 0, 0, err
		} else {
			questions = append(questions, q)
		}
	}
	return questions, startcol, colincrement, nil
}
