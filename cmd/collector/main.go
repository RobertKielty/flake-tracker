package main

import (
	"fmt"
	"os"
	"time"

	ci "github.com/RobertKielty/flake-tracker/pkg/cistatus"
	rep "github.com/RobertKielty/flake-tracker/pkg/report"
	rf "github.com/RobertKielty/flake-tracker/pkg/reportedflake"
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



func main() {
	var startTime = time.Now()
	// TODO this is messed up!
	var ciStatusLogger = setUpLogging("ci-status", startTime)
	var ghLogger = setUpLogging("gh-logger", startTime)

	tgBlocking := &ci.CiStatus{
		Name:        "sig-release-master-informing",
		CollectedAt: startTime,
		Logger:      ciStatusLogger,
	}
	reportedFlake := &rf.ReportedFlake{
		Logger:   ghLogger,
		CiStatus: tgBlocking,
	}
	collectData(tgBlocking, reportedFlake) // TODO ciStatus && reportedFlake need to be decoupled
	rep.RunMarkdownSummaryReport(*tgBlocking)
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
