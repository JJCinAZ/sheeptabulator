package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/360EntSecGroup-Skylar/excelize"
	"io/ioutil"
	"os"
	"regexp"
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
	TotalBonus  int
}

type Member struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Team  string `json:"-"`
}

type PopulationCount struct {
	Freq           int
	OriginalAnswer string
	Bonus          int // If this was a bonus answer, what's the bonus value
}

type Question struct {
	Text             string
	BonusQuestion    bool
	BonusAnswer      string
	BonusValue       int
	PopulationCounts map[string]*PopulationCount
}

var (
	Teams     map[string][]Member // map[team name][]Member
	Questions []Question
	TeamMode  bool
)

func main() {
	var (
		Responses []Response
	)
	individual := flag.Bool("i", false, "Show individual question/answer scores")
	filename := flag.String("f", "", "Spreadsheet with responses to read")
	teamfile := flag.String("teamfile", "", "File name of JSON file with team information (leave off if not using teams)")
	missingMemberMode := flag.String("missing", "avg", "Mode for handling missing members: avg, least, middle")
	printteams := flag.Bool("print", false, "Print Teams")
	flag.Parse()
	if *printteams {
		if err := getTeams(*teamfile); err != nil {
			fmt.Println(err)
			os.Exit(2)
		}
		fmt.Printf("Read %d Teams and %d members from %s\n", len(Teams), totalMembers(), *teamfile)
		if *printteams {
			printTeams()
			os.Exit(0)
		}
	}
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
	Responses, err := readResponses(*filename)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	fmt.Printf("Read %d responses\n", len(Responses))
	Responses = eliminateDups(Responses)
	calcScores(Responses)
	if TeamMode {
		printMissingMembers(Responses)
	}
	printScores(Responses, *individual, *missingMemberMode)
}

// Read responses from XLSX file exported from Microsoft Forms.
// Expect row 1 to have column titles in it with rows 2+ having the data (excel table format)
func readResponses(filename string) ([]Response, error) {
	var (
		startcol, colincrement int
	)
	f, err := excelize.OpenFile(filename)
	if err != nil {
		return nil, err
	}
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return nil, err
	}
	list := make([]Response, 0)
	for rownum, row := range rows {
		var a Response
		if rownum == 0 {
			// First row (row #1), so check titles to see if this spreadsheet is of the expected format
			if Questions, startcol, colincrement, err = checkTitles(row); err != nil {
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

func printMissingMembers(responses []Response) {
	for name, members := range Teams {
		for _, member := range members {
			found := false
			for _, r := range responses {
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

func printScores(responses []Response, individual bool, missingMemberMode string) {
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
		for _, p := range q.PopulationCounts {
			a = append(a, x{Answer: p.OriginalAnswer, PopCount: p.Freq})
		}
		sort.Slice(a, func(i, j int) bool {
			return a[i].PopCount > a[j].PopCount
		})
		for i := range a {
			if q.BonusQuestion && strings.EqualFold(q.BonusAnswer, a[i].Answer) {
				fmt.Printf("\t%3d 🎯\t%s\n", q.BonusValue, a[i].Answer)
			} else {
				fmt.Printf("\t%3d\t%s\n", a[i].PopCount, a[i].Answer)
			}
		}
	}
	if individual {
		// Print Individual Scores
		for idxR := range responses {
			fmt.Println(responses[idxR].Name)
			for i, a := range responses[idxR].Answers {
				fmt.Printf("\t%2d: %3d\t%s\n", i, responses[idxR].AnswerScore[i], a)
			}
			fmt.Printf("\t-----------------------\n\t total %d\n", responses[idxR].TotalScore)
		}
		fmt.Println("")
	}
	sortedScores := make([]memberscore, 0)
	for _, r := range responses {
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
		for _, r := range responses {
			if r.Team == n {
				ts.score += r.TotalScore
				membercount++
			}
		}
		ts.name = n
		ts.members = make([]memberscore, 0, 4)
		for _, r := range responses {
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

func (q *Question) mostFreqAnswer() int {
	max := 0
	for _, v := range q.PopulationCounts {
		if v.Freq > max {
			max = v.Freq
		}
	}
	return max
}

func calcScores(responses []Response) {
	// First create a map for each question with the frequency of each answer
	for _, r := range responses {
		for i, answerText := range r.Answers {
			if len(answerText) > 0 {
				a := strings.ToLower(answerText)
				if pc, exists := Questions[i].PopulationCounts[a]; exists {
					pc.Freq++
				} else {
					Questions[i].PopulationCounts[a] = &PopulationCount{Freq: 1, OriginalAnswer: answerText}
				}
			}
		}
	}

	// Calculate value of bonus answers
	for idxQ := range Questions {
		if Questions[idxQ].BonusQuestion {
			i := Questions[idxQ].mostFreqAnswer()
			Questions[idxQ].BonusValue = i + (i >> 1)
		}
	}

	// Now go through the answers in each response and assign the score to each based on the frequency map
	for idxR := range responses {
		responses[idxR].TotalScore = 0
		for i, a := range responses[idxR].Answers {
			if len(a) > 0 {
				a = strings.ToLower(a)
				score := Questions[i].PopulationCounts[a].Freq
				if Questions[i].BonusQuestion && strings.EqualFold(Questions[i].BonusAnswer, a) {
					score = Questions[i].BonusValue
				}
				responses[idxR].AnswerScore[i] = score
				responses[idxR].TotalScore += score
			}
		}
	}
}

func eliminateDups(responses []Response) []Response {
	// Eliminate duplicate responses by a member, taking the last completed one only
	sort.Slice(responses, func(i, j int) bool {
		if responses[i].Email == responses[j].Email {
			return responses[i].Completed.Before(responses[j].Completed)
		}
		return responses[i].Email < responses[j].Email
	})
	var last Response
	newlist := make([]Response, 0, len(responses))
	for i, a := range responses {
		if i > 0 {
			if a.Email != last.Email {
				newlist = append(newlist, last)
			}
		}
		last = a
	}
	if len(last.Email) > 0 {
		newlist = append(newlist, last)
	}
	return newlist
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

func firstRune(s string) rune {
	var first rune
	for _, r := range s {
		first = r
		break
	}
	return first
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
		if len(q) == 0 {
			return nil, 0, 0, fmt.Errorf("column #%d has no title", i)
		}
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
		if firstRune(q) == '🎯' {
			// Bonus question.  Expect bonus answer to be on the end of the title: "Question [bonus answer]"
			regx := regexp.MustCompile(`(?U)(^.+)\s*\[(.*)\]\s*$`)
			if matches := regx.FindStringSubmatch(q); matches == nil {
				return nil, 0, 0, fmt.Errorf("column #%d is a bonus question but is missing ending answer <%s [answer]>", i, q)
			} else {
				questions = append(questions, Question{
					Text:             matches[1],
					BonusAnswer:      matches[2],
					BonusQuestion:    true,
					PopulationCounts: make(map[string]*PopulationCount),
				})
			}
		} else {
			questions = append(questions, Question{Text: q, PopulationCounts: make(map[string]*PopulationCount)})
		}
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

func printTeams() {
	for name, members := range Teams {
		fmt.Printf("Team Name: %s\n", name)
		for _, member := range members {
			fmt.Printf("\t%s\n", member.Name)
		}
	}

	for _, members := range Teams {
		for _, member := range members {
			fmt.Printf("%s,", member.Email)
		}
	}
	fmt.Println("")
}
