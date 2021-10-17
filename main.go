package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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
		err       error
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
		if err = getTeams(*teamfile); err != nil {
			fmt.Println(err)
			os.Exit(2)
		}
		fmt.Printf("Read %d Teams and %d members from %s\n", len(Teams), totalMembers(), *teamfile)
	} else {
		fmt.Printf("Teams mode is disabled\n")
	}
	Responses, err = readResponses(*filename)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	Responses = eliminateDups(Responses)
	fmt.Printf("Read %d responses\n", len(Responses))
	calcScores(Responses)
	if TeamMode {
		printMissingMembers(Responses)
	}
	printScores(Responses, *individual, *missingMemberMode)
}

// Read responses from input file
func readResponses(filename string) ([]Response, error) {
	switch strings.ToUpper(filepath.Ext(filename)) {
	case ".XLS", ".XLSX":
		return readXLSX(filename)
	case ".CSV":
		return readCSV(filename)
	}
	return nil, fmt.Errorf("invalid file type, must be .xls, .xlsx, or .csv")
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

	// Make a slice of team info with scores, so we can sort it
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

func buildQuestion(text string) (q Question, err error) {
	if firstRune(text) == '🎯' {
		// Bonus question.  Expect bonus answer to be on the end of the title: "Question [bonus answer]"
		regx := regexp.MustCompile(`(?U)(^.+)\s*\[(.*)\]\s*$`)
		if matches := regx.FindStringSubmatch(text); matches == nil {
			err = fmt.Errorf("bonus question is missing ending answer <%s [answer]>", text)
			return
		} else {
			q = Question{
				Text:             matches[1],
				BonusAnswer:      matches[2],
				BonusQuestion:    true,
				PopulationCounts: make(map[string]*PopulationCount),
			}
		}
	} else {
		q = Question{Text: text, PopulationCounts: make(map[string]*PopulationCount)}
	}
	return
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
