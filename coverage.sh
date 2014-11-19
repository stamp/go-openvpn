#!/bin/bash
go test -coverprofile=/tmp/c.out
go tool cover -html=/tmp/c.out -o /var/www/error/coverage.html
