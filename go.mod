module github.com/thockin/go-build-template

go 1.14

require (
	github.com/RobertKielty/flake-traker/pkg/cistatus v0.0.0-00010101000000-000000000000
	github.com/RobertKielty/flake-traker/pkg/reportedflake v0.0.0-00010101000000-000000000000
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/sirupsen/logrus v1.7.0
	github.com/stretchr/testify v1.4.0 // indirect
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43 // indirect
	golang.org/x/sys v0.0.0-20200803210538-64077c9b5642 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
)

replace github.com/RobertKielty/flake-traker/pkg/cistatus => /home/rkielty/go/src/github.com/RobertKielty/flake-tracker/pkg/cistatus

replace github.com/RobertKielty/flake-traker/pkg/reportedflake => /home/rkielty/go/src/github.com/RobertKielty/flake-tracker/pkg/reportedflake
