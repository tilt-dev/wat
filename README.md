# Win At Testing with WAT!

Ever experienced brain freeze when you can’t quite come up with the right test
incantation?

Do you use your IDE to run tests because you can’t be bothered to put together
the regexp for the one test you care about?

Have you ever been stuck waiting on a slow test, and wondered if there was a
faster test that you could be running instead?

When we get frustrated at these moments, we think: `wat`.

## What does it do?

`wat` looks at:

- how long each test takes
- how often each test fails
- what files you’ve recently edited

and runs the tests most likely to fail Right Now.

The more you use it, the more it will learn about which test gives fast,
useful feedback on what you’re working on.

## Installation

If you have Go installed, it's easy to install `wat` from source:

```
go get -u github.com/windmilleng/wat/cmd/wat
```

If you don't have Go installed, we may have a precompiled binary for you on the [releases page](https://github.com/windmilleng/wat/releases).

## Usage

From the root of your project, run

```
wat
```

The first time you run `wat`, it will:

1) detect the test commands you might want to run, then
2) make random changes to your codebase then run the test to see what breaks

`wat` will train itself until you interrupt it by pressing `[Enter]` or `[Esc]`.

Then (and every subsequent run), `wat` will suggest some tests and run them for you. (To see the suggested commands without auto-running them, use `--dry-run`.)

Here's an example of output you might see:

```
$ wat
Beginning training...type <Enter> or <Esc> to interrupt
Running all tests in the current workspace
 37 / 37 [==========================================================================] 100.00% 12s

Fuzzing "cli/wat/wat.go" and running all tests
 24 / 37 [=======================================================>------------------] 66.22% 8s

WAT will run the following commands:
	go test github.com/windmilleng/wat/data/db/dbpath
	go test github.com/windmilleng/wat/os/sysctl
	go test github.com/windmilleng/wat/cli/dirs
--------------------
$ go test github.com/windmilleng/wat/data/db/dbpath
ok  	github.com/windmilleng/wat/data/db/dbpath
--------------------
$ go test github.com/windmilleng/wat/os/sysctl
ok  	github.com/windmilleng/wat/os/sysctl	
--------------------
$ go test github.com/windmilleng/wat/cli/dirs
ok  	github.com/windmilleng/wat/cli/dirs	
```

You can also explicitly kick off training with:
```
wat train
```

## Supported Languages

`wat` supports:

- Go
- JavaScript (with package.json)
- Python (with `pytest`)

We welcome contributions to add more language detection!


## Privacy

This tool can send usage reports to https://events.windmill.build, to help us
understand what features people use. We only report on which `wat` commands
run and how long they run for.

You can enable usage reports by running

```
wat analytics opt in
```

(and disable them by running `wat analytics opt out`.)

We do not report any personally identifiable information. We do not report any
identifiable data about your code.

We do not share this data with anyone who is not an employee of Windmill
Engineering.  Data may be sent to third-party service providers like Datadog,
but only to help us analyze the data.

## License
Copyright 2018 Windmill Engineering

Licensed under [the Apache License, Version 2.0](LICENSE)
