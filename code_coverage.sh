#!/bin/bash
# 1. Run tests across all packages and generate a coverage profile
go test ./... -coverprofile=coverage.out

# 2. Analyze the profile to see function-by-function breakdown and the grand total
go tool cover -func=coverage.out

rm coverage.out