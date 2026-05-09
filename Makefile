.PHONY: test update-golden dev run live build

test:
	go test ./... -count=1

update-golden:
	go test ./... -update

dev:
	find . -name '*.go' | entr -cr make test

run:
	go run . testdata/small.parquet

live:
	find . -name '*.go' | entr -cr go run . testdata/small.parquet

build:
	go build -o parquetv .
