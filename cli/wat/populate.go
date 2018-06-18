package wat

import (
	"context"
	"encoding/json"
	"fmt"

	"time"

	"os"

	"github.com/spf13/cobra"
	"github.com/windmilleng/wat/errors"
)

// This file contains code for both `Populate` and `List`

const listTTL = time.Hour * 48

var populateCmd = &cobra.Command{
	Use:   "populate",
	Short: "Smartly populate a list of available test commands (and associate files)",
	Run:   populate,
}

func populate(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	ws, err := GetOrInitWatWorkspace()
	if err != nil {
		Fatal("GetWatWorkspace", err)
	}

	cmds, err := List(ctx, ws, 0 /* always fresh */)
	if err != nil {
		Fatal("Populate", err)
	}

	fmt.Printf("Successfully populated %d commands.\n", len(cmds.Commands))
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all the commands to choose from",
	Run:   list,
}

func list(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	ws, err := GetOrInitWatWorkspace()
	if err != nil {
		Fatal("GetWatWorkspace", err)
	}

	cmdList, err := List(ctx, ws, listTTL)
	if err != nil {
		Fatal("List", err)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(cmdList)
	if err != nil {
		Fatal("Encode", err)
	}
}

type CommandList struct {
	Timestamp time.Time
	Commands  []WatCommand
}

// Get the list of commands.
//
// If there are already commands on disk fresher than the ttl, return those
// commands. Otherwise, generates a list of available WatCommands (i.e. command
// + associated files) and writes them to disk (.wat/list).
//
// Set the ttl to 0 to always regenerate.
func List(ctx context.Context, ws WatWorkspace, ttl time.Duration) (CommandList, error) {
	if ttl > 0 {
		exists, err := ws.Exists(fnameList)
		if err != nil {
			return CommandList{}, err
		}

		if exists {
			cmdList, err := listFromFile(ws)
			if err != nil {
				return cmdList, err
			}

			if time.Since(cmdList.Timestamp) < ttl {
				// List is current, yay!
				return cmdList, nil
			}
		}
	}

	// List is stale or does not exist, (re)populate
	cmds, err := populateAt(ctx, ws)
	if err != nil {
		return CommandList{}, fmt.Errorf("populateAt: %v", err)
	}

	cmdList := CommandList{
		Timestamp: time.Now(),
		Commands:  cmds,
	}
	err = cmdList.toFile(ws)
	if err != nil {
		return CommandList{}, fmt.Errorf("toFile: %v", err)
	}

	return cmdList, nil
}

func listFromFile(ws WatWorkspace) (cmdList CommandList, err error) {
	listContents, err := ws.Read(fnameList)
	if err != nil {
		return cmdList, fmt.Errorf("read: %v", err)
	}

	err = json.Unmarshal(listContents, &cmdList)
	if err != nil {
		return cmdList, fmt.Errorf("json.Unmarshal: %v", err)
	}
	return cmdList, nil
}

func (cl CommandList) toFile(ws WatWorkspace) error {
	j := MustJson(cl)
	return ws.Write(fnameList, j)
}

type WatCommand struct {
	Command     string
	FilePattern string
}

func (c WatCommand) Empty() bool {
	return c.Command == ""
}

// PrettyCmd returns a string suitable for prettily printing this cmd to terminal
func (c WatCommand) PrettyCmd() string {
	escapedCmd := fmt.Sprintf("%q", c.Command)
	escapedCmd = escapedCmd[1 : len(escapedCmd)-1] // remove quotes
	return TermBold(fmt.Sprintf("$ %s\n", escapedCmd))
}

func populateAt(ctx context.Context, ws WatWorkspace) ([]WatCommand, error) {
	result := RunBuiltinPlugins(ctx, ws)

	userResult, err := RunUserPlugins(ctx, ws)
	if err != nil {
		return result, errors.Propagatef(err, "RunUserPlugins")
	}

	result = append(result, userResult...)

	// TODO: dedupe?
	return result, nil
}
