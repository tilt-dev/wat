package wat

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"strconv"

	"github.com/spf13/cobra"
)

var CmdTimeout time.Duration

const Divider = "--------------------\n"

const appNameWat = "wat"

var dryRun bool
var numCmds int

var rootCmd = &cobra.Command{
	Use:   "wat",
	Short: "WAT (Win At Tests!) figures out what tests you should run next, and runs them for you",
	Run:   wat,
}

func init() {
	rootCmd.PersistentFlags().DurationVarP(&CmdTimeout, "timeout", "t", 2*time.Minute, "timeout for training/running commands")
	rootCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "just print recommended commands, don't run them")
	rootCmd.Flags().IntVarP(&numCmds, "numCmds", "n", nDecideCommands, "number of commands WAT should suggest/run")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(recentCmd)
	rootCmd.AddCommand(populateCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(trainCmd)
}

func Execute() (outerErr error) {
	_, analyticsCmd, err := initAnalytics()
	if err != nil {
		return err
	}

	rootCmd.AddCommand(analyticsCmd)

	return rootCmd.Execute()
}

func wat(_ *cobra.Command, args []string) {
	ctx := context.Background()

	ws, err := GetOrInitWatWorkspace()
	if err != nil {
		ws.Fatal("GetWatWorkspace", err)
	}

	// TODO: should probs be able to pass edits into `Decide` (or use the edits that
	// `Decide` found) rather than needing to get them twice.
	recentEdits, err := RecentFileNames(ws)

	cmds, err := Decide(ctx, ws, numCmds)
	if err != nil {
		ws.Fatal("Decide", err)
	}

	if dryRun {
		fmt.Fprintln(os.Stderr, "WAT recommends the following commands:")
	} else {
		fmt.Fprintln(os.Stderr, "WAT will run the following commands:")
	}
	for _, cmd := range cmds {
		// print recommended cmds to terminal (properly escaped, but not wrapped in quotes,
		// in case user wants to copy/paste, pipe somewhere, etc.)
		safe := fmt.Sprintf("%q", cmd.Command)
		fmt.Printf("\t%s\n", tryUnquote(safe))
	}

	if dryRun {
		// it's a dry run, don't actually run the commands
		return
	}

	logContext := LogContext{
		RecentEdits: recentEdits,
		StartTime:   time.Now(),
		Source:      LogSourceUser,
	}

	err = RunCommands(ctx, ws, cmds, CmdTimeout, os.Stdout, os.Stderr, logContext)
	if err != nil {
		ws.Fatal("RunCommands", err)
	}
}

func runCmdAndLog(ctx context.Context, root string, c WatCommand, outStream, errStream io.Writer) (CommandLog, error) {
	start := time.Now()

	err := runCmd(ctx, root, c.Command, outStream, errStream)

	if ctx.Err() != nil {
		// Propagate cancel/timeout errors
		return CommandLog{}, ctx.Err()
	}

	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			// NOT an exit error, i.e. it's an unexpected error; stop execution.
			return CommandLog{}, err
		}
	}

	// Either we have no error, or an ExitError (i.e. expected case: cmd
	// exited with non-zero exit code).
	return CommandLog{
		Command:  c.Command,
		Success:  err == nil,
		Duration: time.Since(start),
	}, nil
}

func runCmd(ctx context.Context, root, command string, outStream, errStream io.Writer) error {
	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = root
	cmd.Stdout = outStream
	cmd.Stderr = errStream

	return cmd.Run()
}

func runCmds(ctx context.Context, root string, cmds []WatCommand, timeout time.Duration,
	outStream, errStream io.Writer) ([]CommandLog, error) {
	logs := []CommandLog{}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	errStream.Write([]byte(Divider))
	for _, c := range cmds {
		outStream.Write([]byte(c.PrettyCmd()))

		log, err := runCmdAndLog(timeoutCtx, root, c, outStream, errStream)
		if err != nil {
			return logs, err
		}

		errStream.Write([]byte(Divider))
		logs = append(logs, log)
	}

	return logs, nil
}

// Runs the given commands and logs their results to file for use in making our ML model
func RunCommands(ctx context.Context, ws WatWorkspace, cmds []WatCommand, timeout time.Duration,
	outStream, errStream io.Writer, logContext LogContext) error {
	t := time.Now()
	logs, err := runCmds(ctx, ws.Root(), cmds, timeout, outStream, errStream)
	if err != nil {
		// If we got an unexpected err running commands, don't bother logging
		return err
	}
	ws.a.Timer(timerCommandsRun, time.Since(t), nil)
	logGroup := CommandLogGroup{
		Logs:    logs,
		Context: logContext,
	}
	return CmdLogGroupsToFile(ws, []CommandLogGroup{logGroup})
}

func tryUnquote(s string) string {
	res, err := strconv.Unquote(s)
	if err == nil {
		return res
	}
	return s
}
