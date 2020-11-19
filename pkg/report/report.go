package report

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"text/template"
	"time"

	ci "github.com/RobertKielty/flake-tracker/pkg/cistatus"
)

// daysAgo returns number of days since time t
// used for reported entities and states have existed
func daysAgo(t time.Time) int {
	return int(time.Since(t).Hours() / 24)
}

func dashboardJobCount(ciStatus ci.CiStatus) int {
	return len (ciStatus.FlakingJobs) + len(ciStatus.FailedJobs) + len(ciStatus.PassingJobs)
}

// dumpStruct outputs the feilds and values of a struct for debugging template{{dumpStruct $j}}
func dumpStruct(s interface{}) string {
	return fmt.Sprintf("%+v",s)
}

// RunReportFromTemplateFile runs report found in tmplFile using data contained in all ciStatuses 
func RunReportFromTemplateFile( tmplFile string, destination io.Writer, ciStatuses ...ci.CiStatus ) {

	funcMap := template.FuncMap{
		"daysAgo":daysAgo,
		"dashboardJobCount":dashboardJobCount,
		"dumpStruct":dumpStruct,
	}

	var tmplName = "mdTemplate"
	mdTemplate , err := ioutil.ReadFile(tmplFile)
	if err != nil {
		panic(err)
	}

	tmpl, err := template.New(tmplName).Funcs(funcMap).Parse(string(mdTemplate))

	if err != nil {
		log.Fatalf("Error parsing %s : %s", tmplName, err)
	}

	// Run the template to verify the output.
	for _, ciStatus := range ciStatuses {
		err = tmpl.Execute(destination, ciStatus)
		if err != nil {
			log.Fatalf("Error executing report %s", err)
		}
	}

}
