package wat

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const testTimeout = time.Second

var cmdFoo = WatCommand{Command: "echo -n foo; echo ' and foo'"}
var logFoo = CommandLog{Command: cmdFoo.Command, Success: true}
var outputFoo = "foo and foo"
var cmdBar = WatCommand{Command: "echo -n bar; echo ' and bar'"}
var logBar = CommandLog{Command: cmdBar.Command, Success: true}
var outputBar = "bar and bar"
var cmdBaz = WatCommand{Command: "echo -n baz; echo ' and baz'"}
var logBaz = CommandLog{Command: cmdBaz.Command, Success: true}
var outputBaz = "baz and baz"
var cmdFalse = WatCommand{Command: "false"} // will always exit w/ code: 1
var logFalse = CommandLog{Command: cmdFalse.Command, Success: false}
var cmdSleep = WatCommand{Command: "sleep 2"}

func TestRunCmds(t *testing.T) {
	ctx := context.Background()
	cmds := []WatCommand{cmdFoo, cmdBar, cmdBaz}
	outBuf := bytes.Buffer{}
	errBuf := bytes.Buffer{}

	logs, err := runCmds(ctx, ".", cmds, testTimeout, &outBuf, &errBuf)
	if err != nil {
		t.Fatal("runCmds:", err)
	}

	expectedStrs := []string{
		cmdFoo.PrettyCmd(), outputFoo,
		cmdBar.PrettyCmd(), outputBar,
		cmdBaz.PrettyCmd(), outputBaz,
	}
	assertContainsStrings(t, expectedStrs, outBuf.String(), "stdOut")

	expectedLogs := []CommandLog{logFoo, logBar, logBaz}
	assertCommandLogs(t, expectedLogs, logs)
}

func TestRunCmdsFailure(t *testing.T) {
	ctx := context.Background()
	cmds := []WatCommand{cmdFoo, cmdFalse, cmdBaz}
	outBuf := bytes.Buffer{}
	errBuf := bytes.Buffer{}

	logs, err := runCmds(ctx, ".", cmds, testTimeout, &outBuf, &errBuf)
	if err != nil {
		// non-zero exit status should NOT return an err, as this is an expected case
		t.Fatal("runCmds:", err)
	}

	expectedStrs := []string{cmdFoo.PrettyCmd(), outputFoo, cmdFalse.PrettyCmd()}
	assertContainsStrings(t, expectedStrs, outBuf.String(), "stdOut")

	expectedLogs := []CommandLog{logFoo, logFalse, logBaz}
	assertCommandLogs(t, expectedLogs, logs)
}

func TestRunCmdsCapturesStdErr(t *testing.T) {
	ctx := context.Background()
	cmds := []WatCommand{
		WatCommand{Command: "echo 'hello world' > /dev/stderr"},
	}
	outBuf := bytes.Buffer{}
	errBuf := bytes.Buffer{}

	_, err := runCmds(ctx, ".", cmds, testTimeout, &outBuf, &errBuf)
	if err != nil {
		t.Fatal("runCmds:", err)
	}

	assertContainsStrings(t, []string{"hello world"}, errBuf.String(), "stdErr")
}

func TestRunCmdsTimesOut(t *testing.T) {
	ctx := context.Background()
	cmds := []WatCommand{cmdFoo, cmdSleep, cmdBaz}
	outBuf := bytes.Buffer{}
	errBuf := bytes.Buffer{}

	logs, err := runCmds(ctx, ".", cmds, time.Second, &outBuf, &errBuf)
	if err != context.DeadlineExceeded {
		t.Fatalf("Expected a DeadlineExceeded error. Actual: %v", err)
	}

	expectedStrs := []string{
		cmdFoo.PrettyCmd(),
		outputFoo,
		cmdSleep.PrettyCmd(),
	}
	assertContainsStrings(t, expectedStrs, outBuf.String(), "stdOut")

	// We DON'T expect logBaz and logSleep.
	// cmdBaz should not run, b/c in case of timeout we
	// bail on all subsequent commands
	expectedLogs := []CommandLog{logFoo}
	assertCommandLogs(t, expectedLogs, logs)
}

func TestRunCmdsUnexepctedError(t *testing.T) {
	cmds := []WatCommand{cmdFoo}

	// Run cmds inside an already-canceled context so we get an error
	ctx := context.Background()
	ctxCancelled, cancelFn := context.WithCancel(ctx)
	cancelFn()

	logs, err := runCmds(ctxCancelled, ".", cmds, testTimeout, &bytes.Buffer{}, &bytes.Buffer{})

	if err == nil {
		t.Fatal("Expected an error b/c context has been canceled, but no error returned")
	}

	if len(logs) != 0 {
		t.Fatalf("Expected 0 logs, got %d", len(logs))
	}
}

func TestRunCommands(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()
	ws := f.watInit()

	ctx := context.Background()
	cmds := []WatCommand{cmdFoo, cmdBar, cmdBaz}
	outBuf := bytes.Buffer{}
	errBuf := bytes.Buffer{}
	now := time.Now()
	edits := []string{"a", "b", "c"}
	logContext := LogContext{
		RecentEdits: edits,
		StartTime:   now,
		Source:      LogSourceUser,
	}

	err := RunCommands(ctx, ws, cmds, testTimeout, &outBuf, &errBuf, logContext)
	if err != nil {
		t.Fatal("RunCommands:", err)
	}

	expectedLogs := CommandLogGroup{
		Logs:    []CommandLog{logFoo, logBar, logBaz},
		Context: logContext,
	}

	assertCmdLogFileContents(t, expectedLogs)
}

func TestRunCommandsRunErr(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()
	ws := f.watInit()

	ctx := context.Background()
	cmds := []WatCommand{cmdFoo, cmdSleep, cmdBaz}
	err := RunCommands(ctx, ws, cmds, testTimeout, &bytes.Buffer{}, &bytes.Buffer{}, LogContext{})
	if err == nil {
		t.Fatal("Expected an error b/c of bad command, but no error returned")
	}

	if _, err := os.Stat(filepath.Join(kWatDirName, fnameCmdLog)); err == nil {
		t.Fatal("Log file written, but should not have been, since we encountered an error.")
	}
}
