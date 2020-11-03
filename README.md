# Flake Tracker

Creates a point-in-time CSV (for now) report listing tests that produce non-determinstic results (NDRs) found on the Jobs reported as FLAKY in the TestGridSummary for sig-release-blocking and sig-release-informing

The report offers the following benefits :

  * provides automatic on-demand status updates for weekly Release Team Meeting
  
  * shows distribution of NDRs accross the project per job, per SIG 
  
  * TODO shows what NDRs are and are not being tracked by GH Issues 

  * TODO shows distribution of categorised effort (Awaiting response, triaged, PR submitted, monitoring, fixed) accross the project per job, per sig

## Building
The Project is built using Tim Hockin's https://github.com/thockin/go-build-template 

These build instructions are copied accross verbatim from there

Run `make` or `make build` to compile your app.  This will use a Docker image
to build your app, with the current directory volume-mounted into place.  This
will store incremental state for the fastest possible build.  Run `make
all-build` to build for all architectures.

Run `make container` to build the container image.  It will calculate the image
tag based on the most recent git tag, and whether the repo is "dirty" since
that tag (see `make version`).  Run `make all-container` to build containers
for all supported architectures.

Run `make push` to push the container image to `REGISTRY`.  Run `make all-push`
to push the container images for all architectures.

Run `make clean` to clean up.

Run `make help` to get a list of available targets.
``` 
make build
```

## Usage - running locally
The build will create an executable in the bin directory in a folder based on your machine architecture. 

Before running the report you will need to first setup a GITHUB_AUTH_TOKEN

Then you need to add the auth token to your env as a GITHUB_AUTH_TOKEN environment var.

Run a report on sig-release-blocking

``` 
$ export GITHUB_AUTH_TOKEN=INSERT_A_GITHUB_AUTH_TOKEN
$ ./bin/OS_ARCH/collector 2> app.log > report.log
```
where OS_ARCH will be your operating system and hardware architechture

app.log will contaier errors encountered during the report run broadly fallin into the following categories
- errors encountered accessing TestGrid or Github
- errors parsing and extracting names of tests and jobs in Github Issues on the CI Signal project Board

## Parameters and environment ##
At present no parameters are required to run the program future versions may have the following cmd line flags
TODO 
* --config file YAML file that contains report configuration, tabgroups, project boards, output format, datastore
* --gh-token / env var GitHub Oauth2 token
* --tab-group TestGrid TabGroup
* --project-board GithubProjectBoard yaml
* --output Output format - json, csv, org
* --port - if specificed starts a server listenting on port and displays a HTML version of the report
TODO 
I wanted to log output to specific files but that is not working - not urgent

