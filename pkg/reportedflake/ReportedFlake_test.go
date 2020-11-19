package reportedflake

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	ci "github.com/RobertKielty/flake-tracker/pkg/cistatus"
	"github.com/google/go-github/github"
	log "github.com/sirupsen/logrus"
)

func TestSetUp(t *testing.T) {

}

// Tests which hunts for CI Job tests named in a Github Issue under
// the heading Which tests are flaking:
func TestGetReportedTests(t *testing.T) {
	var tests = []struct {
		name        string
		input       string
		testsWanted []string
	}{
		{
			"Single test - correctly reported",
			` Which test(s) are flaking:
[sig-instrumentation] MetricsGrabber should grab all metrics from a Scheduler

Testgrid link:
`,
			// Note: Array iof strings that captures order in which tests are parsed
			[]string{"[sig-instrumentation] MetricsGrabber should grab all metrics from a Scheduler"},
		},
		{
			"Mulitple tests - correctly reported",
			` Which test(s) are flaking:
[sig-instrumentation] MetricsGrabber should grab all metrics from a Scheduler
[sig-network] Services should be able to preserve UDP traffic when server pod cycles for a NodePort service
Testgrid link:
`,
			// Note: Array iof strings that captures order in which tests are parsed
			[]string{
				"[sig-instrumentation] MetricsGrabber should grab all metrics from a Scheduler",
				"[sig-network] Services should be able to preserve UDP traffic when server pod cycles for a NodePort service",
			},
		},
		{
			"Issue from Github",
			"**Which test(s) are failing**:" + "`" + "[sig-api-machinery] CustomResourcePublishOpenAPI [Privileged:ClusterAdmin] works for CRD preserving unknown fields in an embedded object [Conformance]" + "`",
			[]string{
				"[sig-api-machinery] CustomResourcePublishOpenAPI [Privileged:ClusterAdmin] works for CRD preserving unknown fields in an embedded object [Conformance]",
			},
		},
	}
	// Instantiate a Rported Flake
	var rf ReportedFlake
	for _, test := range tests {

		reportedTests, _ := rf.getReportedTests(test.input)

		actualTestCount := len(reportedTests)
		expectedTestCount := len(test.testsWanted)

		if actualTestCount != expectedTestCount {
			t.Logf("For %v", test.name)
			t.Errorf("\texpected to find %d reported test(s) but found %d\n", expectedTestCount, actualTestCount)
			t.Errorf("\treportTests are %v\n", reportedTests)
		}

		for i, reportedTest := range reportedTests {
			if strings.Compare(reportedTest, test.testsWanted[i]) != 0 {
				t.Logf("For %v", test.name)
				t.Errorf("\nexpected reported test was\n%s\nbut found\n%s\n", test.testsWanted[i], reportedTest)
			}
		}
	}
}

func TestReportedFlake_CollectIssuesFromBoard(t *testing.T) {
	type fields struct {
		ghId     string
		repo     string
		job      string
		tests    []string
		Logger   *log.Logger
		CiStatus *ci.CiStatus
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rf := &ReportedFlake{
				ghId:     tt.fields.ghId,
				repo:     tt.fields.repo,
				job:      tt.fields.job,
				tests:    tt.fields.tests,
				Logger:   tt.fields.Logger,
				CiStatus: tt.fields.CiStatus,
			}
			rf.CollectIssuesFromBoard()
		})
	}
}

func TestReportedFlake_linkIssueToFlakingJob(t *testing.T) {
	type fields struct {
		ghId     string
		repo     string
		job      string
		tests    []string
		Logger   *log.Logger
		CiStatus *ci.CiStatus
	}
	type args struct {
		i *github.Issue
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rf := &ReportedFlake{
				ghId:     tt.fields.ghId,
				repo:     tt.fields.repo,
				job:      tt.fields.job,
				tests:    tt.fields.tests,
				Logger:   tt.fields.Logger,
				CiStatus: tt.fields.CiStatus,
			}
			if err := rf.linkIssueToFlakingJob(tt.args.i); (err != nil) != tt.wantErr {
				t.Errorf("ReportedFlake.linkIssueToFlakingJob() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_getCITargets(t *testing.T) {
	type args struct {
		re  *regexp.Regexp
		url string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getCITargets(tt.args.re, tt.args.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("getCITargets() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getCITargets() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReportedFlake_getReportedTests(t *testing.T) {
	type fields struct {
		ghId     string
		repo     string
		job      string
		tests    []string
		Logger   *log.Logger
		CiStatus *ci.CiStatus
	}
	type args struct {
		b string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name "Happyday scenario"
			args "
<!-- Please only use this template for submitting reports about flaky tests or jobs (pass or fail with no underlying change in code) in Kubernetes CI -->

**Which jobs are flaking**:

**Which test(s) are flaking**:

**Testgrid link**:

**Reason for failure**:

**Anything else we need to know**:
- links to go.k8s.io/triage appreciated
- links to specific failures in spyglass appreciated

<!-- Please see the deflaking doc (https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/flaky-tests.md) for more guidance! -->"
want[]
		}
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rf := &ReportedFlake{
				ghId:     "",
				repo:     "",
				job:      "",
				tests:    ,
				Logger:   ,
				CiStatus: ,
			}
			got, err := rf.getReportedTests(tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReportedFlake.getReportedTests() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReportedFlake.getReportedTests() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReportedFlake_removeSrcMarkdown(t *testing.T) {
	tests := []struct {
		name   string
		arg    string
		want   string
	}{
		{
			"Markdown testname is formatted as src using a bactic",
			"`[sig-api-machinery] CustomResourcePublishOpenAPI [Privileged:ClusterAdmin] works for CRD preserving unknown fields in an embedded object [Conformance]`",
			"[sig-api-machinery] CustomResourcePublishOpenAPI [Privileged:ClusterAdmin] works for CRD preserving unknown fields in an embedded object [Conformance]",
		},
		{
			"testname is not formatted",
			"[sig-api-machinery] CustomResourcePublishOpenAPI [Privileged:ClusterAdmin] works for CRD preserving unknown fields in an embedded object [Conformance]",
			"[sig-api-machinery] CustomResourcePublishOpenAPI [Privileged:ClusterAdmin] works for CRD preserving unknown fields in an embedded object [Conformance]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rf := &ReportedFlake{
				ghId:     "",
				repo:     "",
				job:      "",
				tests:    nil,
				Logger:   nil,
				CiStatus: nil,
			}
			if got := rf.removeSrcMarkdown(tt.arg); got != tt.want {
				t.Errorf("ReportedFlake.removeMarkdown() returned\n%v\nwanted\n%v", got, tt.want)
			}
		})
	}
}

func TestReportedFlake_getIssueDetail(t *testing.T) {
	type fields struct {
		ghId     string
		repo     string
		job      string
		tests    []string
		Logger   *log.Logger
		CiStatus *ci.CiStatus
	}
	type args struct {
		client        *github.Client
		jobSummaryUrl string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *github.Issue
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rf := &ReportedFlake{
				ghId:     tt.fields.ghId,
				repo:     tt.fields.repo,
				job:      tt.fields.job,
				tests:    tt.fields.tests,
				Logger:   tt.fields.Logger,
				CiStatus: tt.fields.CiStatus,
			}
			got, err := rf.getIssueDetail(tt.args.client, tt.args.jobSummaryUrl)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReportedFlake.getIssueDetail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReportedFlake.getIssueDetail() = %v, want %v", got, tt.want)
			}
		})
	}
}
