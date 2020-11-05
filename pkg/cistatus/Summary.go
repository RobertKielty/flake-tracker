package cistatus

// Retrieves CI Status from summary report for a named TestGrid TabGroup
// Retrieves data from TestGrid for now
// TODO BigQuery should be the single source of truth)
import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

const (
	TG_TABGROUP_SUMMARY_FMT string = "https://testgrid.k8s.io/%s/summary"
	TG_JOB_TEST_TABLE_FMT   string = "https://testgrid.k8s.io/%s/table?tab=%s&width=5&exclude-non-failed-tests=&sort-by-flakiness=&dashboard=%s"
)

// TabGroupStatus tracks status of CI Jobs for a named TestGrid TabGroup
type CiStatus struct {
	Name               string
	CollectedAt        time.Time
	Count              int
	TabGroupSummaryUrl string
	FlakingJobs        map[string]JobStatus
	PassingJobs        map[string]JobStatus
	FailedJobs         map[string]JobStatus
	Logger             *log.Logger
}

// JobStatus mirrors data on the TestGrid summary status
type JobStatus struct {
	OverallStatus           string             `json:"overall_status"`
	Alert                   string             `json:"alert"`
	LastRun                 int64              `json:"last_run_timestamp"`
	LastUpdate              int64              `json:"last_update_timestamp"`
	LatestGreenRun          string             `json:"latest_green"`
	LatestStatusIcon        string             `json:"overall_status_icon"`
	LatestStatusDescription string             `json:"status"`
	Url                     string             // Url for testGridJobResult
	JobTestResults          *testGridJobResult // See CollectFlakyTests
}

type testGridJobResult struct {
	TestGroupName string `json:"test-group-name"`
	/* - Unused fields from REST query
	           - Retained as comment for possible future use
		           possible future report extention
				Query         string `json:"query"`
				Status        string `json:"status"`
				PhaseTimer    struct {
					Phases []string  `json:"phases"`
					Delta  []float64 `json:"delta"`
					Total  float64   `json:"total"`
				} `json:"phase-timer"`
				Cached  bool   `json:"cached"`
				Summary string `json:"summary"`
				Bugs    struct {
				} `json:"bugs"`
				Changelists       []string   `json:"changelists"`
				ColumnIds         []string   `json:"column_ids"`
				CustomColumns     [][]string `json:"custom-columns"`
				ColumnHeaderNames []string   `json:"column-header-names"`
				Groups            []string   `json:"groups"`
				Metrics           []string   `json:"metrics"`
	*/
	Tests []struct {
		Name         string        `json:"name"`
		OriginalName string        `json:"original-name"`
		Alert        interface{}   `json:"alert"`
		LinkedBugs   []interface{} `json:"linked_bugs"`
		Messages     []string      `json:"messages"`
		ShortTexts   []string      `json:"short_texts"`
		Statuses     []struct {
			Count int `json:"count"`
			Value int `json:"value"`
		} `json:"statuses"`
		Target       string      `json:"target"`
		UserProperty interface{} `json:"user_property"`
		// Calculated Field added here
		Sig string
	} `json:"tests"`
	/*  Remainder of Unused fields
		RowIds       []string    `json:"row_ids"`
		Timestamps   []int64     `json:"timestamps"`
		Clusters     interface{} `json:"clusters"`
		TestIDMap    interface{} `json:"test_id_map"`
		TestMetadata struct {
		} `json:"test-metadata"`
		StaleTestThreshold    int    `json:"stale-test-threshold"`
		NumStaleTests         int    `json:"num-stale-tests"`
		AddTabularNamesOption bool   `json:"add-tabular-names-option"`
		ShowTabularNames      bool   `json:"show-tabular-names"`
		Description           string `json:"description"`
		BugComponent          int    `json:"bug-component"`
		CodeSearchPath        string `json:"code-search-path"`
		OpenTestTemplate      struct {
			URL     string `json:"url"`
			Name    string `json:"name"`
			Options struct {
			} `json:"options"`
		} `json:"open-test-template"`
		FileBugTemplate struct {
			URL     string `json:"url"`
			Name    string `json:"name"`
			Options struct {
				Body  string `json:"body"`
				Title string `json:"title"`
			} `json:"options"`
		} `json:"file-bug-template"`
		AttachBugTemplate struct {
			URL     string `json:"url"`
			Name    string `json:"name"`
			Options struct {
			} `json:"options"`
		} `json:"attach-bug-template"`
		ResultsURLTemplate struct {
			URL     string `json:"url"`
			Name    string `json:"name"`
			Options struct {
			} `json:"options"`
		} `json:"results-url-template"`
		CodeSearchURLTemplate struct {
			URL     string `json:"url"`
			Name    string `json:"name"`
			Options struct {
			} `json:"options"`
		} `json:"code-search-url-template"`
		AboutDashboardURL string `json:"about-dashboard-url"`
		OpenBugTemplate   struct {
			URL     string `json:"url"`
			Name    string `json:"name"`
			Options struct {
			} `json:"options"`
		} `json:"open-bug-template"`
		ContextMenuTemplate struct { // JobStatus{
			URL     string `json:"url"`
			Name    string `json:"name"`
			Options struct {
			} `json:"options"`
		} `json:"context-menu-template"`
	ResultsText   string      `json:"results-text"`
	LatestGreen   string      `json:"latest-green"`
	TriageEnabled bool        `json:"triage-enabled"`
	Notifications interface{} `json:"notifications"`
	OverallStatus int         `json:"overall-status"`
	*/
}

// CollectFlakyTest queries TestGrid for a list of flaking tests for each Job
// that is currently Flaky and adds the tests to
func (t *CiStatus) CollectFlakyTests() error {

	for jobName := range t.FlakingJobs {
		url := fmt.Sprintf(TG_JOB_TEST_TABLE_FMT,
			t.Name, url.QueryEscape(jobName), t.Name)

		resp, err := http.Get(url)

		if err != nil {
			t.Logger.Error("HTTP get job test results", err, url)
			return err
		}

		body, err := ioutil.ReadAll(resp.Body)
		var flakingTestResults testGridJobResult
		err = json.Unmarshal(body, &flakingTestResults)
		if err != nil {
			t.Logger.Error("Unmarshalling Test Result", err, url)
			return err
		}
		// SearchLoggedIssues()
		// Store data and url where we found it. tmp var used as per
		// https://github.com/golang/go/issues/3117#issuecomment-66063615
		var tmp = t.FlakingJobs[jobName]
		addSigToTestResults(&flakingTestResults)
		tmp.JobTestResults = &flakingTestResults
		tmp.Url = url
		t.FlakingJobs[jobName] = tmp
	}
	return nil
}

// CollectFailedTests adds a list of Tests that are failing for each Failed Job
func (t *CiStatus) CollectFailedTests() error {

	for jobName := range t.FailedJobs {
		url := fmt.Sprintf(TG_JOB_TEST_TABLE_FMT,
			t.Name, url.QueryEscape(jobName), t.Name)
		resp, err := http.Get(url)
		if err != nil {
			t.Logger.Error("HTTP get Failed job test results", err, url)
			return err
		}
		body, err := ioutil.ReadAll(resp.Body)
		var failedTestResults testGridJobResult
		err = json.Unmarshal(body, &failedTestResults)
		if err != nil {
			t.Logger.Error("Unmarshalling Failed Test Result", err, url)
			return err
		}
		// Store data and url where we found it. tmp var used as per
		// https://github.com/golang/go/issues/3117#issuecomment-66063615
		var tmp = t.FailedJobs[jobName]
		addSigToTestResults(&failedTestResults)
		tmp.JobTestResults = &failedTestResults
		tmp.Url = url
		t.FailedJobs[jobName] = tmp
	}
	return nil
}

// addSigToTestResults sets the sig field on tgJobResult using the test name
// by finding the first occurance of [sig-SIGNAME], if no sig is found sets sig
// to "job-owner"
func addSigToTestResults(tgJobResult *testGridJobResult) {
	var sigRe = regexp.MustCompile(`\[sig-.+?\] `)
	for i, t := range tgJobResult.Tests {
		sig := sigRe.FindString(t.Name)
		if sig != "" {
			tgJobResult.Tests[i].Sig = sig
		} else {
			tgJobResult.Tests[i].Sig = "job-owner"
		}
	}
	return
}

// CollectStatus populates t with job status summary data from TestGrid
func (t *CiStatus) CollectStatus() error {

	t.TabGroupSummaryUrl = fmt.Sprintf(TG_TABGROUP_SUMMARY_FMT, t.Name)

	resp, err := http.Get(t.TabGroupSummaryUrl)
	if err != nil {
		t.Logger.Error("HTTP getting", err)
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Logger.Error("Reading HTTP response buffer", err)
		return err
	}

	jobs := make(map[string]JobStatus)

	err = json.Unmarshal(body, &jobs)
	if err != nil {
		t.Logger.Error("UnMarshalling reponse body", err)
		return err
	}

	t.FlakingJobs = make(map[string]JobStatus, 0)

	for name, job := range jobs {
		if job.OverallStatus == "FLAKY" {
			t.FlakingJobs[name] = jobs[name]
		}
	}

	t.FailedJobs = make(map[string]JobStatus, 0)
	for name, job := range jobs {
		if job.OverallStatus == "FAILING" {
			t.FailedJobs[name] = job
		}
	}

	t.PassingJobs = make(map[string]JobStatus, 0)
	for name, job := range jobs {
		if job.OverallStatus == "PASSING" {
			t.PassingJobs[name] = job
		}
	}

	t.Count = len(jobs)
	return nil
}
