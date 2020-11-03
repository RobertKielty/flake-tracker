module github.com/RobertKielty/flake-tracker/pkg/reportedflake

go 1.14

require (
	github.com/RobertKielty/flake-traker/pkg/cistatus v0.0.0-00010101000000-000000000000
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/sirupsen/logrus v1.7.0
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43
)

replace github.com/RobertKielty/flake-traker/pkg/cistatus => /home/rkielty/go/src/github.com/RobertKielty/flake-tracker/pkg/cistatus
