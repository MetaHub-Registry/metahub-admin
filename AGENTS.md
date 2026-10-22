# Agent Instructions

## Build and Test

```sh
./test.sh        # go test -short -coverprofile coverage.txt
go build ./...
```

A change is complete when `./test.sh` passes.
