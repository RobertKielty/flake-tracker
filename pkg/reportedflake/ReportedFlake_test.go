package reportedflake

import "testing"

const (
	FR_TEST = ` Which test(s) are flaking:
[sig-instrumentation] MetricsGrabber should grab all metrics from a Scheduler

Testgrid link:
`
)

// Tests ParseTest which hunts for CI Job tests named in a Github Issue under
// the heading Which tests are flaking:
// As this is a test about tests we refer to the test cases covered here as
// scenarios
func TestParseTest(t *testing.T) {
	var expectedTestCount = 1
	tests, _ := ParseTests(FR_TEST)
	for i, test := range tests {
		t.Logf("Test[%d] is %v", i, test)
	}
	if len(tests) != expectedTestCount {
		t.Errorf("Expected to find %d test(s) but found %d %v \n", expectedTestCount, len(tests), tests)
	}
}
