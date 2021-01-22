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
	Teams     map[string][]Member // map[team name][]Member
	Responses []Response
	Questions []Question
	TeamMode  bool
)

func main() {
	individual := flag.Bool("i", false, "Show individual question/answer scores")
	filename := flag.String("f", "", "Spreadsheet with responses to read")
	teamfile := flag.String("teamfile", "", "File name of JSON file with team information (leave off if not using teams)")
	missingMemberMode := flag.String("missing", "avg", "Mode for handling missing members: avg, least, middle")
	flag.Parse()
	if len(*filename) == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}
	switch *missingMemberMode {
	case "avg", "least", "middle":
	default:
		fmt.Println("-missing must be 'avg', 'least', or 'middle'")
		os.Exit(1)
	}
	if len(*teamfile) > 0 {
		TeamMode = true
	}
	if TeamMode {
		if err := getTeams(*teamfile); err != nil {
			fmt.Println(err)
			os.Exit(2)
		}
		fmt.Printf("Read %d Teams and %d members from %s\n", len(Teams), totalMembers(), *teamfile)
	} else {
		fmt.Printf("Teams mode is disabled\n")
	}
	err := readResponses(*filename)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	fmt.Printf("Read %d responses\n", len(Responses))
	eliminateDups()
	calcScores()
	if TeamMode {
		printMissingMembers()
	}
	printScores(*individual, *missingMemberMode)
}

func readResponses(filename string) error {
	var (
		startcol, colincrement int
	)
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
			if Questions, startcol, colincrement, err = checkTitles(row); err != nil {
				return err
			}
		} else {
			a.Email = strings.ToLower(row[3])
			a.Name = row[4]
			if TeamMode {
				if a.Team, err = findTeam(a.Email); err != nil {
					return err
				}
			}
			f, err := strconv.ParseFloat(row[2], 64)
			if err != nil {
				return fmt.Errorf("invalid Completed time (%s) on row:\n%+v\n", err, row)
			}
			a.Completed = TimeFromExcelTime(f, false)
			for i := 0; i < len(Questions); i++ {
				colIdx := i*colincrement + startcol
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

func printMissingMembers() {
	for name, members := range Teams {
		for _, member := range members {
			found := false
			for _, r := range Responses {
				if strings.EqualFold(r.Email, member.Email) {
					found = true
					break
				}
			}
			if !found {
				fmt.Printf("Missing response from %s on team %s\n", member.Email, name)
			}
		}
	}
}

func printScores(individual bool, missingMemberMode string) {
	type memberscore struct {
		name, email string
		score       int
	}

	// Print Questions and stack-ranked answers
	for i, q := range Questions {
		fmt.Printf("Question #%d -- %s\n", i+1, q.Text)
		// PopulationCounts is a map, so let's make it into an array so we can sort it
		type x struct {
			Answer   string
			PopCount int
		}
		a := make([]x, 0, len(q.PopulationCounts))
		for answer, count := range q.PopulationCounts {
			a = append(a, x{Answer: answer, PopCount: count})
		}
		sort.Slice(a, func(i, j int) bool {
			return a[i].PopCount > a[j].PopCount
		})
		for i := range a {
			fmt.Printf("\t%3d\t%s\n", a[i].PopCount, a[i].Answer)
		}
	}
	if individual {
		// Print Individual Scores
		for idxR := range Responses {
			fmt.Println(Responses[idxR].Name)
			for i, a := range Responses[idxR].Answers {
				fmt.Printf("\t%2d: %3d\t%s\n", i, Responses[idxR].AnswerScore[i], a)
			}
			fmt.Printf("\t-----------------------\n\t total %d\n", Responses[idxR].TotalScore)
		}
		fmt.Println("")
	}
	sortedScores := make([]memberscore, 0)
	for _, r := range Responses {
		sortedScores = append(sortedScores, memberscore{name: r.Name, email: r.Email, score: r.TotalScore})
	}
	sort.Slice(sortedScores, func(i, j int) bool {
		return sortedScores[i].score > sortedScores[j].score
	})
	fmt.Println("\nPlayer Scores")
	for _, m := range sortedScores {
		fmt.Printf("%4d\t%s\n", m.score, m.name)
	}
	if !TeamMode {
		return
	}

	// Make a slice of team info with scores so we can sort it
	type teamscore struct {
		name    string
		score   int
		members []memberscore
	}
	sortedTeams := make([]teamscore, 0, len(Teams))
	for n := range Teams {
		var ts teamscore
		membercount := 0
		for _, r := range Responses {
			if r.Team == n {
				ts.score += r.TotalScore
				membercount++
			}
		}
		ts.name = n
		ts.members = make([]memberscore, 0, 4)
		for _, r := range Responses {
			if r.Team == n {
				ts.members = append(ts.members, memberscore{
					name:  r.Name,
					email: r.Email,
					score: r.TotalScore,
				})
			}
		}
		sort.Slice(ts.members, func(i, j int) bool {
			return ts.members[i].score > ts.members[j].score
		})
		fillinscore := 0
		if membercount > 0 && membercount < 4 {
			switch missingMemberMode {
			case "avg":
				fillinscore = ts.score / membercount
			case "least":
				fillinscore = ts.members[len(ts.members)-1].score
			case "middle":
				switch len(ts.members) {
				case 0:
					fillinscore = 0
				case 1:
					fillinscore = ts.members[0].score
				case 2, 3:
					fillinscore = ts.members[1].score
				}
			}
			ts.score += fillinscore * (4 - membercount)
			for ; membercount < 4; membercount++ {
				ts.members = append(ts.members, memberscore{score: fillinscore, name: "--------"})
			}
		}
		sortedTeams = append(sortedTeams, ts)
	}
	sort.Slice(sortedTeams, func(i, j int) bool {
		return sortedTeams[i].score > sortedTeams[j].score
	})

	// Print team scores
	fmt.Println("\nTeam Scores")
	for _, ts := range sortedTeams {
		fmt.Printf("%4d\t%s\n", ts.score, ts.name)
		for _, m := range ts.members {
			fmt.Printf("\t%4d\t%s\n", m.score, m.name)
		}
	}
}

func calcScores() {
	// First create a map for each question with the frequency of each answer
	for _, r := range Responses {
		for i, a := range r.Answers {
			if len(a) > 0 {
				a = strings.ToLower(a)
				Questions[i].PopulationCounts[a] = Questions[i].PopulationCounts[a] + 1
			}
		}
	}
	// Now go through the answers in each response and assign the score to each based on the frequency map
	for idxR := range Responses {
		score := 0
		for i, a := range Responses[idxR].Answers {
			if len(a) > 0 {
				a = strings.ToLower(a)
				Responses[idxR].AnswerScore[i] = Questions[i].PopulationCounts[a]
				score += Questions[i].PopulationCounts[a]
			}
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

func checkTitles(row []string) ([]Question, int, int, error) {
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
		q := row[i]
		if colincrement > 1 {
			if i+2 >= len(row) {
				return nil, 0, 0, fmt.Errorf("too few columns, expect: answer, points, feedback tuples")
			}
			x := "Points - " + q
			if row[i+1] != x && strings.HasPrefix(row[i+1], "Column") == false {
				return nil, 0, 0, fmt.Errorf("column #%d was supposed to be '%s' but is '%s'", i+1+1, x, row[i+1])
			}
			x = "Feedback - " + q
			if row[i+2] != x && strings.HasPrefix(row[i+2], "Column") == false {
				return nil, 0, 0, fmt.Errorf("column #%d was supposed to be '%s' but is '%s'", i+2+1, x, row[i+2])
			}
		}
		questions = append(questions, Question{Text: q, PopulationCounts: make(map[string]int)})
	}
	return questions, startcol, colincrement, nil
}

func getTeams(filename string) error {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(b, &Teams); err == nil {
		for teamName, members := range Teams {
			for i := range members {
				members[i].Email = strings.ToLower(members[i].Email)
				members[i].Team = teamName
			}
		}
	}
	return err
}
