package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// Read responses from CSV file exported from Google Forms or Google Sheets.
// Expect row 1 to have column titles in it with rows 2+ having the data
func readCSV(filename string) ([]Response, error) {
	var (
		startcol, colincrement, namecol int
		f                               *os.File
		err                             error
		rows                            [][]string
	)
	if f, err = os.Open(filename); err != nil {
		return nil, err
	}
	defer f.Close()
	reader := csv.NewReader(f)
	if rows, err = reader.ReadAll(); err != nil {
		return nil, err
	}
	list := make([]Response, 0)
	for rownum, row := range rows {
		var a Response
		if rownum == 0 {
			// First row (row #1), so check titles to see if this spreadsheet is of the expected format
			if Questions, startcol, colincrement, namecol, err = checkCSVTitles(row); err != nil {
				return nil, err
			}
		} else {
			a.Completed, _ = time.Parse("01/02/2006 14:04:05", row[0])
			a.Email = strings.ToLower(row[1])
			a.Name = a.Email
			if namecol > 0 {
				a.Name = row[namecol]
			}
			//a.Name, _ = getGoogleName(a.Email)
			if TeamMode {
				if a.Team, err = findTeam(a.Email); err != nil {
					return nil, err
				}
			}
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

func checkCSVTitles(row []string) ([]Question, int, int, int, error) {
	var (
		requiredColumns = []string{"Timestamp", "Email Address"}
		namecol         = -1
	)
	if len(row) < 3 {
		return nil, 0, 0, 0, fmt.Errorf("CSV cannot be correct -- has less than 3 columns")
	}
	questions := make([]Question, 0)
	// Trim spaces from column titles
	for i := range row {
		row[i] = strings.TrimSpace(row[i])
	}
	// Check for required columns
	for i, c := range requiredColumns {
		if row[i] != c {
			return nil, 0, 0, 0, fmt.Errorf("column #%d is not %s", i+1, c)
		}
	}
	// Check to see if the first column is "Your Name..."
	startcol, colincrement := 2, 1
	if strings.HasPrefix(row[startcol], "Your Name") {
		startcol++
		namecol = 2
	}
	// Gather questions
	for i := startcol; i < len(row); i += colincrement {
		if len(row[i]) == 0 {
			return nil, 0, 0, 0, fmt.Errorf("column #%d has no title", i)
		}
		row[i] = trimNumberPrefix(row[i])
		if q, err := buildQuestion(row[i]); err != nil {
			return nil, 0, 0, 0, err
		} else {
			questions = append(questions, q)
		}
	}
	return questions, startcol, colincrement, namecol, nil
}

// trimNumberPrefix removes a number prefix from a string
// This function does not work with unicode space characters
func trimNumberPrefix(s string) string {
	var regex = regexp.MustCompile(`^\s*\d+\.\s*`)
	if loc := regex.FindStringIndex(s); loc != nil {
		return s[loc[1]:]
	}
	return s
}
