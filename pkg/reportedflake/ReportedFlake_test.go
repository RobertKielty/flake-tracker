package reportedflake

import (
	"strings"
	"testing"
)

// Tests ParseTest which hunts for CI Job tests named in a Github Issue under
// the heading Which tests are flaking:
func TestParseTest(t *testing.T) {
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
	}

	for _, test := range tests {
		t.Logf("For %v", test.name)

		reportedTests, _ := ParseTests(test.input)

		actualTestCount := len(reportedTests)
		expectedTestCount := len(test.testsWanted)

		if actualTestCount != expectedTestCount {
			t.Errorf("\texpected to find %d reported test(s) but found %d\n", expectedTestCount, actualTestCount)
			t.Errorf("\treportTests are %v\n", reportedTests)
		}

		for i, reportedTest := range reportedTests {
			if strings.Compare(reportedTest, test.testsWanted[i]) != 0 {
				t.Errorf("\nexpected reported test was\n%s\nbut found\n%s\n", test.testsWanted[i], reportedTest)
			}
		}
	}
}
