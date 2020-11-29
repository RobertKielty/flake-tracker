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

	ci "github.com/RobertKielty/flake-tracker/pkg/cistatus"
	"github.com/google/go-github/github"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

const (
	FLAKE_TEMPLATE_HEADER_WTAF      string = "**Which test(s) are flaking**:"
	FAIL_TEMPLATE_HEADER_SWHIBD     string = "**Since when has it been failing**:"
	FLAKE_TEMPLATE_HEADER_TGL       string = "**Testgrid link**:"
	ciSignalBoardId                 int64  = 2093513
	ciSignalNewCardColId            int64  = 4212817
	ciSignalUnderInvestigationColId int64  = 4212819
	ciSignalObservingColId          int64  = 4212821
	TG_MISSING                      string = "missing"
)

var (
	tgUrl         = regexp.MustCompile(`https://testgrid.k8s.io/.+`)
	tgDashboardRE = regexp.MustCompile(`o/(.+)#`)
	tgJobRE       = regexp.MustCompile(`#(.+)`)
)

// ReportedFlake - issue logged on Github for a test that produces non-deterministic results
type ReportedFlake struct {
	ghId, repo, job string
	tests           []string
	Logger          *log.Logger
	CiStatus        *ci.CiStatus
}

// Decoration of a Reported Issue, ghIssue with flake-related data extracted from issue comments
type actualFlake struct {
	ghIssue   *github.Issue
	dashboard string
	job       string
	tests     []string
}

// linkIssueToFlakingJob extracts flake-related data (testnames and jobnames) from the first comment in a GitHub issue
// adding it to t.FlakeIssues[job].jobTestResults.Tests.LinkedBugs map key'd by CI job and where named tests match.
func (rf *ReportedFlake) linkIssueToFlakingJob(i *github.Issue) error {
	tgLink := tgUrl.FindString(*i.Body)

	if len(tgLink) > 0 {
		d, err := getCITargets(tgDashboardRE, tgLink)
		if err != nil {
			return errors.New("linkIssueToFlakingJob : Error decorating issue " + err.Error())
		}
		j, err := getCITargets(tgJobRE, tgLink)
		if err != nil {
			return errors.New("linkIssueToFlakingJob : Error decorating issue " + err.Error())
		}
		//		rf.Logger.Debugf("linkIssueToFlakingJob : Searching for mentioned tests in :%s", *i.Body)
		reportedTests, err := rf.getReportedTests(i.GetBody()) // Getting tests from initial body for now; may need to process comments aswel
		if err != nil {
			return errors.New("linkIssueToFlakingJob : Error decorating issue " + strconv.FormatInt(*i.ID, 10))
		}

		rf.Logger.Debugf("linkIssueToFlakingJob : Issue has mentioned these tests :%v", reportedTests)

		// Append this report to the list of flakes logged against this job
		var tmp actualFlake
		tmp.dashboard = d
		tmp.job = j
		tmp.ghIssue = i
		tmp.tests = reportedTests
		if j != "" {
			// check that j is a valid job
			if rf.CiStatus.IsValidJob(j) {
				job, exists := rf.CiStatus.FlakingJobs[j]
				if exists {
					rf.Logger.Infof("linkIssueToFlakingJob: Linking Reported Issue : %s %s\n", d, j)
					for _, test := range job.JobTestResults.Tests {
						test.LinkedBugs = make([]interface{}, len(reportedTests))
						for _, reportedTest := range reportedTests {
							if test.Name == reportedTest {
								rf.Logger.Infof("linkIssueToFlakingJob: JOIN! %s == %s \n", test.Name, reportedTest)
								test.LinkedBugs = append(test.LinkedBugs, tmp)
							} else {
								rf.Logger.Infoln("linkIssueToFlakingJob: NO_JOIN!")
								rf.Logger.Infof("%12s:%s", "test.Name", test.Name)
								rf.Logger.Infof("%12s:%s", "reportedTest", reportedTest)
							}
						}
					}
				} else {
					// TODO else statment can be removed in time
					rf.Logger.Debugf("linkIssueToFlakingJob: job %s not reported here", j)
				}
			} else {
				rf.Logger.Errorf("linkIssueToFlakingJob: Invalid Job Lookup %s := getCITargets(%s,%s)", j, tgJobRE.String(), tgLink)
				rf.Logger.Errorf("linkIssueToFlakingJob: job present as %s", rf.CiStatus.GetJobStatus(j))
				rf.Logger.Errorf("linkIssueToFlakingJob: PassingJobs %s", rf.CiStatus.GetJobsByStatus("PASSING"))
				rf.Logger.Errorf("linkIssueToFlakingJob: FailedJobs %s", rf.CiStatus.GetJobsByStatus("FAILED"))
				rf.Logger.Errorf("linkIssueToFlakingJob: FlakingJobs %s", rf.CiStatus.GetJobsByStatus("FLAKING"))

				return errors.New("linkIssueToFlakingJob: Job " + j + " not valid " + *i.Title + " " + i.GetHTMLURL())
			}
		}

	} else {
		return errors.New("linkIssueToFlakingJob:Could not find TestGridURL regExp in Issue" + *i.Title)
	}
	return nil
}

// CollectIssuesFromBoard retrieves Reported Flakes from issues present on the CI Signal Board
// and adds them to the LinkedBugs[] on the jobTestResults
func (rf *ReportedFlake) CollectIssuesFromBoard() {

	githubApiToken := os.Getenv("GITHUB_AUTH_TOKEN")
	if githubApiToken == "" {
		rf.Logger.Error("GITHUB_API_TOKEN is not exported in process env.")
		panic("Quitting GITHUB_API_TOKEN not exported in process env")
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

	if err != nil {
		rf.Logger.Error(err)
		rf.Logger.Error(r)
		panic("Github client could not get")
	}

	var totalCardCound = 0
	for _, col := range cols {
		rf.Logger.Infof(" ColName : %s", col.GetName())
		cards, _, err := client.Projects.ListProjectCards(ctx, *col.ID, opt)

		if err != nil {
			rf.Logger.Error(err)
			rf.Logger.Error(r)
			panic("Github client could not get")
		}
		var colCardCount = 0
		for _, card := range cards {
			contentUrl := card.GetContentURL()
			if contentUrl != "" { // is this preventing me from calling linkIssueToFlakingJob!! WHY??
				rf.Logger.Debugf("contentUrl : %s", contentUrl)
				// colId := card.GetColumnID() if (colId == ciSignalNewCardColId) || (colId == ciSignalUnderInvestigationColId) || (colId == ciSignalObservingColId) {
				issue, err := rf.getIssueDetail(client, contentUrl)
				rf.Logger.Debugf("issueTitle : %s", issue.GetTitle())
				if err != nil {
					rf.Logger.Errorf("getIssueDetail(%s) returned %v\n", card.GetContentURL(), err)
					break
				}
				err = rf.linkIssueToFlakingJob(issue)
				if err != nil {
					rf.Logger.Errorf("Error linking Flake for this card %s, %s\nReason: %v", issue.GetTitle(), issue.GetURL(), err)
					break
				}
			} else {
				rf.Logger.Warnf("WE WANT TO EXIT? contentUrl empty for %v",card)
				os.Exit(100)
			}
			colCardCount++
		}
		totalCardCound ++
		rf.Logger.Infof("    Visited %d cards on col ", colCardCount, col.GetName())
	}
	rf.Logger.Infof("Visited %d cards in total", totalCardCound)
}

// getCITargets is a helper that returns first occurance of the group in re in the string url
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
// is considered to be a reported test
func (rf *ReportedFlake) getReportedTests(b string) ([]string, error) {
	var tests []string
	rf.Logger.Debugf("getReportedTests test string: \"%v\"",b)
        // TODO This is not working?
	start := strings.Index(b, FLAKE_TEMPLATE_HEADER_WTAF) + len(FLAKE_TEMPLATE_HEADER_WTAF)
	end := strings.Index(b, FLAKE_TEMPLATE_HEADER_TGL) // - len(FLAKE_TEMPLATE_HEADER_TGL)

	if start == -1 {
		rf.Logger.Debugf("getReportedTests: start is -1 scanning b[%d:%d]\n",start,end)
		rf.Logger.Debugf("getReportedTests: header missing %s",FLAKE_TEMPLATE_HEADER_WTAF)
		return nil, errors.New("Missing header " + FLAKE_TEMPLATE_HEADER_WTAF)
	}

	if end == -1 {
		rf.Logger.Debugf("getReportedTests end is -1 scanning b[%d:%d]\n",start,end)
		rf.Logger.Debugf("getReportedTests: header missing %s",FLAKE_TEMPLATE_HEADER_TGL)
		return nil, errors.New("Missing header " + FLAKE_TEMPLATE_HEADER_TGL)
	}

	// Start reading after "Which test(s) are flaking:"
	r := strings.NewReader(b[start:end])
	s := bufio.NewScanner(r)
	rf.Logger.Debugf("getReportedTests scanning b[%d:%d] %v\n",start,end, b[start:end])
	for s.Scan() {
		t := s.Text()
		ct := rf.removeSrcMarkdown(t)
		if len(ct) > 1 {
			rf.Logger.Debugf("getReportedTests adding %v\n", ct)
			tests = append(tests, ct)
		}
	}
	rf.Logger.Debugf("getReportedTests scanned tests %v\n", tests)
	return tests, nil
}

// removeSrcMarkdown trims surrounding backtics from string s if present.
// If you are interested in why this looks as gak as it does, see
// https://github.com/golang/go/issues/32590
// https://www.fileformat.info/info/unicode/char/0060/index.htm
func (rf *ReportedFlake) removeSrcMarkdown(s string) string {
	return strings.TrimFunc(s, func(r rune) bool {
		return (r == 0x0060) // backtic Unicode in hex
	})
}

// getIssueDetail takes gh client and an contentUrl from a gh card and returns a gh issue
func (rf *ReportedFlake) getIssueDetail(client *github.Client, contentUrl string) (*github.Issue, error) {
	rf.Logger.Tracef("getIssueDetail %s\n", contentUrl)
	if contentUrl == "" {
		err := errors.New("jobSummary url is nil")
		return nil, err
	}
	urlParts := strings.Split(contentUrl, "/")
	if len(urlParts) < 3 {
		err := errors.New(fmt.Sprintf("Error spliting url %s", contentUrl))
		return nil, err
	}

	i := urlParts[len(urlParts)-1]
	r := urlParts[len(urlParts)-3]
	o := urlParts[len(urlParts)-4]

	rf.Logger.Tracef("getIssueDetail i:%s r:%s o:%s\n", i,r,o)
	issueNumber, err := strconv.Atoi(i)
	if err != nil {
		return nil, err
	}
	ghIssue, _, err := client.Issues.Get(context.Background(), o, r, issueNumber)
	rf.Logger.Tracef("getIssueDetail %s", ghIssue.GetURL() )

	if err != nil {
		return nil, err
	}
	return ghIssue, nil
}
