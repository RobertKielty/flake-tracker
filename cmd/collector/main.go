package main

import (
	"fmt"
	"os"
	"time"

	ci "github.com/RobertKielty/flake-traker/pkg/cistatus"
	rf "github.com/RobertKielty/flake-traker/pkg/reportedflake"
	log "github.com/sirupsen/logrus"
)

// TODO Create a Flake Issue linter module for use by this and Prow Robot
// TODO Create a Presenter package
// TODO Replace TestGrid scraping with BigTable Queries

var (
	reportFields log.Fields
)

func collectData(cs *ci.CiStatus, rf *rf.ReportedFlake) {
	log.SetFormatter(&log.TextFormatter{})
	reportFields = log.Fields{
		"DATA BEING RETRIEVED": "Job Status Summary TestGrid TabGroup",
		"TEST_GRID TAB_GROUP":  cs.Name,
		"COLLECTION TIME":      cs.CollectedAt,
		"TB GRP SMMRY URL":     cs.TabGroupSummaryUrl,
	}

	cs.CollectStatus()
	cs.CollectFlakyTests()
	cs.CollectFailedTests()
	rf.CollectIssuesFromBoard(cs)
}

// runs a report against on the jobs in tab group status summary for jobs
// that are flaking, failed and passing
func runReport(cs *ci.CiStatus) {

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

func main() {
	var startTime = time.Now()
	// TODO this is messed up!
	var ciStatusLogger = setUpLogging("ci-status", startTime)
	var ghLogger = setUpLogging("gh-logger", startTime)

	tgBlocking := &ci.CiStatus{
		Name:        "sig-release-master-blocking",
		CollectedAt: startTime,
		Logger:      ciStatusLogger,
	}
	reportedFlake := &rf.ReportedFlake{
		Logger:   ghLogger,
		CiStatus: tgBlocking,
	}
	collectData(tgBlocking, reportedFlake) // TODO ciStatus && reportedFlake need to be decoupled
	runReport(tgBlocking)                  // TODO extract runReport to a reporter class parameterised on format
	tgBlocking.Logger.Writer().Close()
}

func setUpLogging(name string, startTime time.Time) *log.Logger {

	var (
		formattedTime = startTime.Format("Jan-02-2006")
		logFilename   = fmt.Sprintf("%s-%s.log", name, formattedTime)
		logger        = log.New()
	)

	logger.SetFormatter(&log.JSONFormatter{})
	logger.SetLevel(log.TraceLevel)

	// For now, one human readable log file with datetime stamp per run
	file, err := os.OpenFile(logFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)

	if err == nil {
		logger.Out = file
	} else {
		logger.Error("Failed to log to file, using default stderr", err)
	}

	return logger
}
