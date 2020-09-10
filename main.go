package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/360EntSecGroup-Skylar/excelize"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Response struct {
	Email       string
	Name        string
	Team        string
	Completed   time.Time
	Answers     []string
	AnswerScore []int
	TotalScore  int
}

type Member struct {
	Email string `json:"email"`
	Team  string `json:"-"`
}

type Question struct {
	Text             string
	PopulationCounts map[string]int
}

var (
	Teams     map[string][]Member
	Responses []Response
	Questions []Question
)

func main() {
	individual := flag.Bool("i", false, "Show individual scores")
	filename := flag.String("f", "", "Spreadsheet with responses to read")
	flag.Parse()
	if len(*filename) == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}
	if err := getTeams(); err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	fmt.Printf("Read %d Teams and %d members\n", len(Teams), totalMembers())
	err := readResponses(*filename)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	fmt.Printf("Read %d responses\n", len(Responses))
	eliminateDups()
	calcScores()
	printScores(*individual)
}

func readResponses(filename string) error {
	f, err := excelize.OpenFile(filename)
	if err != nil {
		return err
	}
	Responses = make([]Response, 0)
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return err
	}
	for rownum, row := range rows {
		var a Response
		if rownum == 0 {
			if Questions, err = checkTitles(row); err != nil {
				return err
			}
		} else {
			a.Email = strings.ToLower(row[3])
			a.Name = row[4]
			if a.Team, err = findTeam(a.Email); err != nil {
				return err
			}
			f, err := strconv.ParseFloat(row[2], 64)
			if err != nil {
				return fmt.Errorf("invalid Completed time (%s) on row:\n%+v\n", err, row)
			}
			a.Completed = TimeFromExcelTime(f, false)
			for i := 0; i < len(Questions); i++ {
				colIdx := i*3 + 7
				if colIdx < len(row) {
					a.Answers = append(a.Answers, strings.TrimSpace(row[colIdx]))
				}
			}
			a.AnswerScore = make([]int, len(Questions))
			Responses = append(Responses, a)
		}
	}
	return nil
}

func printScores(individual bool) {
	// Print Questions and stack-ranked answers
	for i, q := range Questions {
		fmt.Printf("Question #%d -- %s\n", i+1, q.Text)
		for answer, count := range q.PopulationCounts {
			fmt.Printf("\t%d\t%s\n", count, answer)
		}
	}
	if individual {
		// Print Individual Scores
		for idxR := range Responses {
			fmt.Println(Responses[idxR].Name)
			for i, a := range Responses[idxR].Answers {
				fmt.Printf("\t%2d: %d\t%s\n", i, Responses[idxR].AnswerScore[i], a)
			}
			fmt.Printf("\t-----------------------\n\t total %d\n", Responses[idxR].TotalScore)
		}
	}
	// Print team scores
	fmt.Println("\nTeam Scores")
	for n, _ := range Teams {
		teamscore := 0
		for _, r := range Responses {
			if r.Team == n {
				teamscore += r.TotalScore
			}
		}
		fmt.Printf("%s\t%d\n", n, teamscore)
		for _, r := range Responses {
			if r.Team == n {
				fmt.Printf("\t%d\t%s\n", r.TotalScore, r.Name)
			}
		}

	}
}

func calcScores() {
	// First create a map for each question with the frequency of each answer
	for _, r := range Responses {
		for i, a := range r.Answers {
			a = strings.ToLower(a)
			Questions[i].PopulationCounts[a] = Questions[i].PopulationCounts[a] + 1
		}
	}
	// Now go through the answers in each response and assign the score to each based on the frequency map
	for idxR := range Responses {
		score := 0
		for i, a := range Responses[idxR].Answers {
			a = strings.ToLower(a)
			Responses[idxR].AnswerScore[i] = Questions[i].PopulationCounts[a]
			score += Questions[i].PopulationCounts[a]
		}
		Responses[idxR].TotalScore = score
	}
}

func eliminateDups() {
	// Eliminate duplicate responses by a member, taking the last completed one only
	sort.Slice(Responses, func(i, j int) bool {
		if Responses[i].Email == Responses[j].Email {
			return Responses[i].Completed.Before(Responses[j].Completed)
		}
		return Responses[i].Email < Responses[j].Email
	})
	var last Response
	a2 := make([]Response, 0, len(Responses))
	for i, a := range Responses {
		if i > 0 {
			if a.Email != last.Email {
				a2 = append(a2, last)
			}
		}
		last = a
	}
	if len(last.Email) > 0 {
		a2 = append(a2, last)
	}
	Responses = a2
}

func totalMembers() int {
	m := 0
	for _, v := range Teams {
		m += len(v)
	}
	return m
}

func findTeam(email string) (string, error) {
	for _, v := range Teams {
		for _, m := range v {
			if m.Email == email {
				return m.Team, nil
			}
		}
	}
	return "", fmt.Errorf("cannot find '%s' on any team", email)
}

func checkTitles(row []string) ([]Question, error) {
	var cols = []string{"ID", "Start time", "Completion time", "Email", "Name", "Total points", "Quiz feedback"}
	questions := make([]Question, 0)
	for i, c := range cols {
		if row[i] != c {
			return nil, fmt.Errorf("column #%d is not %s", i+1, c)
		}
	}
	for i := 7; i < len(row); i += 3 {
		q := row[i]
		x := "Points - " + q
		if row[i+1] != x {
			return nil, fmt.Errorf("column #%d was supposed to be '%s' but is '%s'", i+1+1, x, row[i+1])
		}
		x = "Feedback - " + q
		if row[i+2] != x {
			return nil, fmt.Errorf("column #%d was supposed to be '%s' but is '%s'", i+2+1, x, row[i+2])
		}
		questions = append(questions, Question{Text: q, PopulationCounts: make(map[string]int)})
	}
	return questions, nil
}

func getTeams() error {
	b, err := ioutil.ReadFile("teams.json")
	if err != nil {
		return err
	}
	if err = json.Unmarshal(b, &Teams); err == nil {
		for k, v := range Teams {
			for i, _ := range v {
				v[i].Team = k
			}
		}
	}
	return err
}
