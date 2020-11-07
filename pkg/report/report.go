package report

import (
	"fmt"
	"time"
	ci "github.com/RobertKielty/flake-tracker/pkg/cistatus"
)

// runs a report against on the jobs in tab group status summary for jobs
// that are flaking, failed and passing
func RunReport(cs *ci.CiStatus) {

	reportStartTime := cs.CollectedAt.Format(time.UnixDate)
	flakeCount := 0
	failCount := 0
	passingCount := 0

	for jobName, job := range cs.FlakingJobs {
		flakeCount++
		results := job.JobTestResults
		for i, flakyTest := range results.Tests {
			// jobOwner,
			if len(flakyTest.LinkedBugs) > 0 {
				for _, reportedBy := range flakyTest.LinkedBugs {
					fmt.Printf(`%s,%s,%s,"%d of %d","%s","%s","%s","%T - %v"`+"\n",
						reportStartTime,
						job.OverallStatus,
						jobName,
						i+1,
						len(results.Tests),
						flakyTest.Name,
						job.Url,
						flakyTest.Sig,
						reportedBy,
						reportedBy)
				}
			} else {
				fmt.Printf(`%s,%s,%s,"%d of %d","%s","%s","%s"`+"\n",
					reportStartTime,
					job.OverallStatus,
					jobName,
					i+1,
					len(results.Tests),
					flakyTest.Name,
					job.Url,
					flakyTest.Sig)
			}
		}
	}

	for jobName, jobStatus := range cs.FailedJobs {
		failCount++
		jobFailedTests := jobStatus.JobTestResults
		for _, failedTest := range jobFailedTests.Tests {
			fmt.Printf("%s,%s,%s,\"%s\",\"%s\",%s\n",
				cs.CollectedAt.Format(time.UnixDate),
				jobStatus.OverallStatus, jobName, failedTest.Sig,
				failedTest.Name, jobStatus.Url)
		}
	}

	for jobName, jobStatus := range cs.PassingJobs {
		passingCount++
		fmt.Printf("%s,%s,%s,\"%s\",\"%s\",%s\n",
			cs.CollectedAt.Format(time.UnixDate),
			jobStatus.OverallStatus, jobName, "", "", jobStatus.Url)
	}
	totalCount := flakeCount + failCount + passingCount

	// TODO Summary report
	// Overview, Percentages and Status (RYG) calculation
	fmt.Printf("\"%s\",Total/Failing/Flaking/Passing, %d,%d,%d,%d\n",
		reportStartTime, totalCount, failCount, flakeCount, passingCount)
	fmt.Printf("\"%s\", Percentage Failing/Flaking/Passing, %2.1f ,%2.1f ,%2.1f\n",
		reportStartTime,
		float64(failCount)/float64(totalCount)*100,
		float64(flakeCount)/(float64(totalCount))*100,
		float64(passingCount)/float64(totalCount)*100)

}
