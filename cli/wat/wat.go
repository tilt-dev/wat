package wat

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
)

var CmdTimeout time.Duration

const Divider = "--------------------\n"

var rootCmd = &cobra.Command{
	Use:   "wat",
	Short: "Wat tells you what test to run next",
	Run:   wat,
}

func init() {
	rootCmd.PersistentFlags().DurationVarP(&CmdTimeout, "timeout", "t", 2*time.Minute, "Timeout for running commands in WAT")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(recentCmd)
	rootCmd.AddCommand(populateCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(decideCmd)
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

func wat(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	ws, err := GetOrInitWatWorkspace()
	if err != nil {
		ws.Fatal("GetWatWorkspace", err)
	}

	// TODO: should probs be able to pass edits into `Decide` (or use the edits that
	// `Decide` found) rather than needing to get them twice.
	recentEdits, err := RecentFileNames(ws)

	cmds, err := Decide(ctx, ws)
	// TODO(dbentley): grab amount of data to put into recEvent to analyze how data affects usage
	if err != nil {
		ws.Fatal("Decide", err)
	}

	fmt.Println("WAT recommends the following commands:")
	for _, cmd := range cmds {
		fmt.Printf("\t%q\n", cmd.Command)
	}

	// t := time.Now()
	fmt.Println("Run them? [Y/n]")

	ch, err := getChar()
	if err != nil {
		ws.Fatal("getChar", err)
	}
	// ANALYTICS: log timer (user latency)

	runIt := UserYN(ch, true)
	if !runIt {
		fmt.Println("OK, suit yourself!")
		return
	}

	tags := map[string]string{tagAccepted: fmt.Sprintf("%t", runIt)}
	ws.a.Incr(statRecommendation, tags)

	logContext := LogContext{
		RecentEdits: recentEdits,
		StartTime:   time.Now(),
		Source:      LogSourceUser,
	}

	// t = time.Now()
	err = RunCommands(ctx, ws, cmds, CmdTimeout, os.Stdout, os.Stderr, logContext)
	if err != nil {
		ws.Fatal("RunCommands", err)
	}
	// ANALYTICS: log timer (run latency)
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

	outStream.Write([]byte(Divider))
	for _, c := range cmds {
		outStream.Write([]byte(c.PrettyCmd()))

		log, err := runCmdAndLog(timeoutCtx, root, c, outStream, errStream)
		if err != nil {
			return logs, err
		}

		outStream.Write([]byte(Divider))
		logs = append(logs, log)
	}

	return logs, nil
}

// Runs the given commands and logs their results to file for use in making our ML model
func RunCommands(ctx context.Context, ws WatWorkspace, cmds []WatCommand, timeout time.Duration,
	outStream, errStream io.Writer, logContext LogContext) error {
	logs, err := runCmds(ctx, ws.Root(), cmds, timeout, outStream, errStream)
	if err != nil {
		// If we got an unexpected err running commands, don't bother logging
		return err
	}
	logGroup := CommandLogGroup{
		Logs:    logs,
		Context: logContext,
	}
	return CmdLogGroupsToFile(ws, []CommandLogGroup{logGroup})
}
