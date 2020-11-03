package reportedflake

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	ci "github.com/RobertKielty/flake-traker/pkg/cistatus"
	"github.com/google/go-github/github"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

const (
	FLAKE_TEMPLATE_HEADER_WTAF      string = "Which test(s) are flaking:"
	FLAKE_TEMPLATE_HEADER_TGL       string = "Testgrid link:"
	ciSignalBoardId                 int64  = 2093513
	ciSignalNewCardColId            int64  = 4212817
	ciSignalUnderInvestigationColId int64  = 4212819
	ciSignalObservingColId          int64  = 4212821
	TG_MISSING                      string = "missing"
)

var (
	tgUrl    = regexp.MustCompile(`https://testgrid.k8s.io/.+`)
	tgDashRE = regexp.MustCompile(`o/(.+)#`)
	tgJobRE  = regexp.MustCompile(`#(.+)`)
)

// ReportedFlake - issue logged on Github for a test that produces non-deterministic results
type ReportedFlake struct {
	ghId, repo, job string
	tests           []string
	Logger          log.Logger
	ciStatus        ci.CiStatus
}

// Decoration of a GH Issue with flake-related data extracted from issue comments
type actualFlake struct {
	ghIssue   *github.Issue
	dashboard string
	job       string
	tests     []string
}

// parseTests collects tests referenced in the body of a formatted Flake Issue on GitHub
// Each non-empty line between "Which test(s) are flaking:" and Testgrid link:
// is congetTestssidered to be a test
func ParseTests(b string) ([]string, error) {
	var tests []string

	start := strings.Index(b, FLAKE_TEMPLATE_HEADER_WTAF) + len(FLAKE_TEMPLATE_HEADER_WTAF)
	end := strings.Index(b, FLAKE_TEMPLATE_HEADER_TGL)

	if start == -1 {
		return nil, errors.New("Missing header " + FLAKE_TEMPLATE_HEADER_WTAF)
	}

	if end == -1 {
		return nil, errors.New("Missing header " + FLAKE_TEMPLATE_HEADER_TGL)
	}

	// Start reading after "Which test(s) are flaking:"
	r := strings.NewReader(b[start:end])
	s := bufio.NewScanner(r)
	for s.Scan() {
		t := s.Text()
		if len(t) > 1 { // TODO this guards against empty line matches nicer way anyone?
			tests = append(tests, t)
		}
	}
	return tests, nil
}

// decorateFlakeIssue extracts flake-related data from a GitHub issue adding it to
// t.FlakeIssues[job].jobTestResults.Tests.LinkedBugs  map key'd by CI jobName
func (rf *ReportedFlake) decorateFlakeIssue(i *github.Issue) error {
	tgLink := tgUrl.FindString(*i.Body)
	rf.Logger.Debugf("len(tgLin):%d", len(tgLink))
	if len(tgLink) > 0 {
		d, err := getCITargets(tgDashRE, tgLink)
		if err != nil {
			return errors.New("Error decorating issue " + err.Error())
		}
		j, err := getCITargets(tgJobRE, tgLink)
		if err != nil {
			return errors.New("Error decorating issue " + err.Error())
		}

		ta, err := rf.getReportedTests(*i.Body) // Getting tests from initial body for now may need to process comments aswel
		if err == nil {
			return errors.New("Error decorating issue " + strconv.FormatInt(*i.ID, 10))
		}
		rf.Logger.Debugf("Issue has mentioned these tests :%v", ta)
		// Append this report to the list of flakes logged against this job
		var tmp actualFlake
		tmp.dashboard = d
		tmp.job = j
		tmp.ghIssue = i
		tmp.tests = ta
		if j != "" {
			// TODO figure out how the class collaborate!
			// pass in a ref to the cisignal summary object so we can do this lookup
			job, exists := rf.ciStatus.FlakingJobs[j]
			if exists {
				for _, test := range job.JobTestResults.Tests {
					test.LinkedBugs = make([]interface{}, len(ta))
					for _, actualFlake := range ta {
						if test.Name == actualFlake {
							test.LinkedBugs = append(test.LinkedBugs, actualFlake)
						}
					}
				}
			}
		}

	} else {
		return errors.New("Could not find TestGridURL regExp in Issue" + *i.Title)
	}
	return nil
}

// CollectIssuesFromBoard retrieves logged Flake Issues from the CI Signal Board
// and adds them to the LinkedBugs[] on the jobTestResults
func (rf *ReportedFlake) CollectIssuesFromBoard() {

	githubApiToken := os.Getenv("GITHUB_AUTH_TOKEN")
	if githubApiToken == "" {
		rf.Logger.Error("GITHUB_API_TOKEN is not set in process env.")
		panic("Quitting")
	}
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: githubApiToken})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	rl, _, e := client.RateLimits(ctx)

	if _, ok := e.(*github.RateLimitError); ok {
		rf.Logger.Error(rl)
		panic("Github client Rate Limit reached")
	}

	opt := &github.ProjectCardListOptions{}
	listOpt := &github.ListOptions{}
	cols, r, err := client.Projects.ListProjectColumns(ctx, ciSignalBoardId, listOpt)
	rf.Logger.Infof("c.P.LPC cols %v\n", cols)

	if err != nil {
		rf.Logger.Error(err)
		rf.Logger.Error(r)
		panic("Github client could not get")
	}

	for _, col := range cols {
		cards, _, err := client.Projects.ListProjectCards(ctx, *col.ID, opt)

		if err != nil {
			rf.Logger.Error(err)
			rf.Logger.Error(r)
			panic("Github client could not get")
		}

		for _, card := range cards {
			contentUrl := card.GetContentURL()
			if contentUrl != "" {
				rf.Logger.Debugf("card url is :%s", contentUrl)
				// colId := card.GetColumnID()
				// if (colId == ciSignalNewCardColId) || (colId == ciSignalUnderInvestigationColId) || (colId == ciSignalObservingColId) {
				issue, err := rf.getIssueDetail(client, contentUrl)
				rf.Logger.Debugf("issueDetail is :%s", issue.GetTitle())
				if err != nil {
					rf.Logger.Errorf("getIssueDetail(%s) %v returned %v\n",
						card.GetContentURL(), card, err)
					break
				}
				err = rf.decorateFlakeIssue(issue)
				if err != nil {
					rf.Logger.Errorf("Error decorating Flake for this card %s, %s\nReason: %v",
						issue.GetTitle(), issue.GetURL(), err)
					break
				}
				// } else {
				// 	rf.Logger.Warnf("Ignoring card not new or under investigation",card.GetContentURL())			}
			}

		}
	}
}

// getCITargets is a helper that returns first occurance of the group in re
// and err if regexp does not match
func getCITargets(re *regexp.Regexp, url string) (string, error) {
	matches := re.FindStringSubmatch(url)
	if len(matches) > 0 {
		target := strings.TrimSuffix(matches[1], "\r")
		return target, nil
	} else {
		errMsg := fmt.Sprintf("getCITargest RegEx %v not found in url:%s", re.String(), url)
		err := errors.New(errMsg)
		return "", err
	}
}

// getReportedTests collects tests referenced in the body of a formatted Flake Issue on GitHub
// Each non-empty line between "Which test(s) are flaking:" and Testgrid link:
// is congetTestssidered to be a test
func (rf *ReportedFlake) getReportedTests(b string) ([]string, error) {
	var tests []string

	start := strings.Index(b, FLAKE_TEMPLATE_HEADER_WTAF) + len(FLAKE_TEMPLATE_HEADER_WTAF)
	end := strings.Index(b, FLAKE_TEMPLATE_HEADER_TGL)

	if start == -1 {
		return nil, errors.New("Missing header " + FLAKE_TEMPLATE_HEADER_WTAF)
	}

	if end == -1 {
		return nil, errors.New("Missing header " + FLAKE_TEMPLATE_HEADER_TGL)
	}

	// Start reading after "Which test(s) are flaking:"
	r := strings.NewReader(b[start:end])
	s := bufio.NewScanner(r)
	for s.Scan() {
		tests = append(tests, s.Text())
	}
	return tests, nil
}

func (rf *ReportedFlake) getIssueDetail(client *github.Client, jobSummaryUrl string) (*github.Issue, error) {
	rf.Logger.Tracef("getIssueDetail %s\n", jobSummaryUrl)
	if jobSummaryUrl == "" {
		err := errors.New("jobSummary url is nil")
		return nil, err
	}
	urlParts := strings.Split(jobSummaryUrl, "/")
	if len(urlParts) < 3 {
		err := errors.New(fmt.Sprintf("Error spliting url %s", jobSummaryUrl))
		return nil, err
	}

	i := urlParts[len(urlParts)-1]
	r := urlParts[len(urlParts)-3]
	o := urlParts[len(urlParts)-4]

	issueNumber, err := strconv.Atoi(i)
	if err != nil {
		return nil, err
	}
	ghIssue, _, err := client.Issues.Get(context.Background(), o, r, issueNumber)

	if err != nil {
		return nil, err
	}
	return ghIssue, nil
}
