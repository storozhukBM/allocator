GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

COVERAGENAME=coverage.out

all: clean test build

build:
	$(GOBUILD) ./...

buildInlineBounds:
	$(GOBUILD) -gcflags='-m -d=ssa/check_bce/debug=1' ./...

test:
	$(GOTEST) ./...

testDebug:
	$(GOTEST) -v ./...

coverage: clean
	$(GOTEST) -coverpkg=./... -coverprofile=$(COVERAGENAME) ./lib/arena/... && go tool cover -html=$(COVERAGENAME)

clean:
	$(GOLEAN)
	rm -f $(COVERAGENAME)