#!/bin/sh

set -e

GCCGO=gccgo
GOOS=solaris
GOARCH=amd64

INC=golibs/src

subpackages=""

compile_go_package () {
	module="$1"
	shift
	files=""

	for f in $*
	do
		files="$files $INC/$module/$f"
	done

	set -x
	$GCCGO -I $INC -c -o $INC/$module.o $files
	set +x

	subpackages="$subpackages $INC/$module.o"
}

make version.go

compile_go_package code.google.com/p/gcfg/token \
	token.go position.go serialize.go
compile_go_package code.google.com/p/gcfg/scanner \
	scanner.go errors.go
compile_go_package code.google.com/p/gcfg \
	gcfg.go bool.go read.go scanenum.go set.go
compile_go_package github.com/abbot/go-http-auth \
	auth.go basic.go digest.go md5crypt.go misc.go users.go
compile_go_package github.com/ogier/pflag \
	flag.go
compile_go_package github.com/snabb/smtp \
	smtp.go auth.go
compile_go_package zfswatcher.damicon.fi/notifier \
	notifier.go

set -x
$GCCGO -I $INC -o zfswatcher \
	zfswatcher.go leds.go util.go version.go webserver.go \
	setup.go zparse.go \
	osutil_$GOOS.go \
	$subpackages

# eof
