package wat

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"
	"github.com/windmilleng/wat/os/ospath"
)

// The maximum number of commands that decide should return.
// In the future, this might be specified by a flag.
const nDecideCommands = 3

// The extra weight of new duration data, to ensure new data
// isn't drowned out by old data.
// Should be a float64 between 0.0 and 0.5, not inclusive.
// We guarantee that a new piece of data will never have less than this weight.
const newCostExtraWeight = 0.2

// The extra weight to add if successCount or failCount is zero
const failProbabilityZeroCase = 0.1

// TODO(nick): Maybe this should be called 'select'? seems more programmery. Ask others what they think.
var decideCmd = &cobra.Command{
	Use:   "decide",
	Short: "Decide what test to run, given the recently edited files, a list of commands, and a history log",
	Run:   decide,
}

func decide(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	ws, err := GetOrInitWatWorkspace()
	if err != nil {
		ws.Fatal("GetWatWorkspace", err)
	}

	result, err := Decide(ctx, ws)
	if err != nil {
		ws.Fatal("Decide", err)
	}

	for _, cmd := range result {
		fmt.Println(cmd.Command)
	}
}

func Decide(ctx context.Context, ws WatWorkspace) ([]WatCommand, error) {
	t := time.Now()
	cmdList, err := List(ctx, ws, listTTL)
	if err != nil {
		return nil, fmt.Errorf("populateAt: %v", err)
	}

	files, err := ws.WalkRoot()
	if err != nil {
		return nil, fmt.Errorf("walkDir: %v", err)
	}

	cmds := cmdList.Commands
	logGroups, err := Train(ctx, ws, cmds, trainTTL)
	if err != nil {
		return nil, fmt.Errorf("train: %v", err)
	}

	sort.Sort(sort.Reverse(fileInfos(files)))
	res := decideWith(cmds, logGroups, files, nDecideCommands)
	ws.a.Timer(timerDecide, time.Since(t), nil)
	return res, nil
}

// Choose the top N commands to run.
//
// Delegates out to an appropriage algorithm.
//
// cmds: The list of commands to decide from
// logGroups: The history of runs
// files: The list of files in this workspace, in sorted order from most
//    recently modified
func decideWith(cmds []WatCommand, logGroups []CommandLogGroup, files []fileInfo, n int) []WatCommand {
	ds := newDecisionStore()
	ds.AddCommandLogGroups(logGroups)

	// pick the most likely to fail given recent edits.
	return gainDecideWith(cmds, ds, files, n)
}

// Choose the top N commands with the highest gain.
func gainDecideWith(cmds []WatCommand, ds DecisionStore, files []fileInfo, n int) (result []WatCommand) {
	// TODO(nick): Right now, we only use the most recently edited file.
	// There might be other conditions that make more sense, like 3 most-recent.
	mostRecentFile := ""
	if len(files) > 0 {
		mostRecentFile = files[0].name
	}

	if len(cmds) == 0 {
		return cmds
	}

	remainder := append([]WatCommand{}, cmds...)
	cond := Condition{EditedFile: mostRecentFile}
	for len(result) < n && len(remainder) > 0 {
		// Find the maximum-gain test in the remainder list.
		max := remainder[0]
		maxGain := ds.CostSensitiveGain(max, cond)

		// More than one index may have the same cost.
		maxIndices := []int{0}

		for i := 1; i < len(remainder); i++ {
			cmd := remainder[i]
			gain := ds.CostSensitiveGain(cmd, cond)
			if gain > maxGain {
				max = cmd
				maxIndices = []int{i}
				maxGain = gain
			} else if gain == maxGain {
				maxIndices = append(maxIndices, i)
			}
		}

		// Grab all the commands with the same maximum gain-per-cost.
		group := []WatCommand{}
		for _, idx := range maxIndices {
			group = append(group, remainder[idx])
		}

		// If they're enough to satisfy the request, grab all of them.
		// Otherwise, only grab the first one.
		if len(group)+len(result) < n {
			group = group[:1]
			maxIndices = maxIndices[:1]
		}

		// Remove from the remainder array in reverse order,
		// so that the removals don't affect later indices.
		for j := len(maxIndices) - 1; j >= 0; j-- {
			idx := maxIndices[j]
			remainder = append(remainder[:idx], remainder[idx+1:]...)
		}

		// Use the second-tier sort to sort the commands that have the same priority.
		group = secondTierDecideWith(group, ds, files, n)
		result = append(result, group...)

		// On the next iteration of the loop, find the best test command Y
		// given that the current test command X succeeded.
		cond = cond.WithSuccess(group[0].Command)
	}

	if len(result) > n {
		result = result[:n]
	}

	return result
}

// All the "dumb" deciding (the non-ML deciding)
func secondTierDecideWith(cmds []WatCommand, ds DecisionStore, files []fileInfo, n int) (results []WatCommand) {
	// first, decide only based on recency.
	recencyResults, cmds := recencyDecideWith(cmds, files, n)
	results = append(results, recencyResults...)
	if len(results) >= n {
		return results
	}

	// if we don't have enough results, try picking the cheapest commands
	cheapestResults, cmds := cheapestDecideWith(cmds, ds, n-len(results))
	results = append(results, cheapestResults...)
	if len(results) >= n {
		return results
	}

	// if we still don't have enough results, naively pick the first commands.
	naiveResults := naiveDecideWith(cmds, n-len(results))
	return append(results, naiveResults...)
}

// Choose the top N commands to run.
//
// This is a super-simple version that just looks at commands associated with recently
// edited files.
//
// cmds: The list of commands to decide from
// files: The list of files in this workspace, in sorted order from most
//    recently modified.
//
// Returns two sets: the commands we chose, and the commands left.
// This makes it easy to chain with other decision algorithms.
func recencyDecideWith(cmds []WatCommand, files []fileInfo, n int) (result []WatCommand, remainder []WatCommand) {
	result = make([]WatCommand, 0, n)

	// We're going to modify the command array, so we need to clone it first.
	remainder = append([]WatCommand{}, cmds...)

	for _, f := range files {
		for i, cmd := range remainder {
			// TODO(nick): Maybe ospath should have a utility for memoizing parsing of
			// patterns? This is probably not worth optimizing tho.
			matcher, err := ospath.NewMatcherFromPattern(cmd.FilePattern)
			if err != nil {
				continue
			}

			if !matcher.Match(f.name) {
				continue
			}

			result = append(result, cmd)
			if len(result) >= n {
				return result, remainder
			}

			// Remove commands from the array, so that we don't
			// re-consider it on future iterations.
			remainder = append(remainder[:i], remainder[i+1:]...)

			// Move onto the next file
			break
		}
	}

	return result, remainder
}

// Choose the top N commands to run.
//
// This chooses the cheapest command to run.
//
// Returns two sets: the commands we chose, and the commands left.
// This makes it easy to chain with other decision algorithms.
func cheapestDecideWith(cmds []WatCommand, ds DecisionStore, n int) (result []WatCommand, remainder []WatCommand) {
	sorter := WatCommandCostSort{DS: ds}
	for _, c := range cmds {
		if ds.HasCost(c) {
			sorter.Commands = append(sorter.Commands, c)
		} else {
			remainder = append(remainder, c)
		}
	}
	sort.Sort(sorter)

	// Pick the N cheapest commands.
	if n > len(sorter.Commands) {
		n = len(sorter.Commands)
	}
	result = append(result, sorter.Commands[:n]...)
	remainder = append(remainder, sorter.Commands[n:]...)
	return result, remainder
}

// Naively pick the first n commands from the list.
func naiveDecideWith(cmds []WatCommand, n int) []WatCommand {
	if n > len(cmds) {
		n = len(cmds)
	}
	return cmds[:n]
}

type DecisionStore struct {
	costs   map[string]CostEstimate
	history map[CommandWithCondition]ResultHistory
}

func (s DecisionStore) HasCost(cmd WatCommand) bool {
	return s.costs[cmd.Command].Count != 0
}

func (s DecisionStore) Cost(cmd WatCommand) time.Duration {
	return s.costs[cmd.Command].Duration
}

// A gain metric. Currently expressed as a unit of gain / cost
// Gain is directly proportional to failure probability, as explained in the design doc.
// Cost is expressed in seconds
// We weight gain higher than cost as gain ^ 2 / cost
func (s DecisionStore) CostSensitiveGain(cmd WatCommand, cond Condition) float64 {
	dur := s.costs[cmd.Command].Duration
	gain := s.FailureProbability(cmd, cond)
	return gain * gain / dur.Seconds()
}

func (s DecisionStore) FailureProbability(cmd WatCommand, cond Condition) float64 {
	results, ok := s.history[CommandWithCondition{Command: cmd.Command, Condition: cond}]
	if !ok {
		ancestors := cond.Ancestors()
		for _, a := range ancestors {
			results, ok = s.history[CommandWithCondition{Command: cmd.Command, Condition: a}]
			if ok {
				break
			}
		}
	}

	zeroCase := failProbabilityZeroCase

	// If the user is editing a file related to this command
	// (as described by FilePattern), boost the zero case way up.
	editedFile := cond.EditedFile
	cmdPattern := cmd.FilePattern
	if editedFile != "" && cmdPattern != "" {
		matcher, err := ospath.NewMatcherFromPattern(cmdPattern)
		if err == nil && matcher.Match(editedFile) {
			zeroCase = 1
		}
	}

	fail := float64(results.FailCount)
	success := float64(results.SuccessCount)
	if fail == 0 {
		fail = zeroCase
	}
	if success == 0 {
		success = zeroCase
	}
	return fail / (fail + success)
}

func (s DecisionStore) addCommandCost(l CommandLog, ctx LogContext) {
	s.costs[l.Command] = s.costs[l.Command].Add(l, ctx)
}

// Add the history of successes and failures for command against a specific environment condition.
// The condition must NOT express recent edits, because that information is expressed in LogContext.
func (s DecisionStore) addCommandHistory(l CommandLog, ctx LogContext, cond Condition) {
	if cond.EditedFile != "" {
		panic("Called addCommandHistory with malformed condition")
	}

	// Increment the history in the null condition where there are no recently changed files.
	cmdWithCond := CommandWithCondition{Command: l.Command, Condition: cond}
	history := s.history[cmdWithCond]
	s.history[cmdWithCond] = history.Add(l.Success)

	for _, recent := range ctx.RecentEdits {
		// Increment the history in the condition where a file has been edited recently.
		cmdWithCond.Condition = cond.WithEditedFile(recent)
		history := s.history[cmdWithCond]
		s.history[cmdWithCond] = history.Add(l.Success)
	}
}

func (s DecisionStore) AddCommandLogGroup(g CommandLogGroup) {
	logs := g.Logs
	ctx := g.Context

	for i, log := range logs {
		s.addCommandCost(log, ctx)
		s.addCommandHistory(log, ctx, Condition{})

		// Build up correlations between commands.
		for j := i + 1; j < len(g.Logs); j++ {
			logJ := g.Logs[j]
			if log.Success {
				s.addCommandHistory(logJ, ctx, Condition{}.WithSuccess(log.Command))
			}

			if logJ.Success {
				s.addCommandHistory(log, ctx, Condition{}.WithSuccess(logJ.Command))
			}
		}
	}

}

func (s DecisionStore) AddCommandLogGroups(logGroups []CommandLogGroup) {
	for _, g := range logGroups {
		s.AddCommandLogGroup(g)
	}
}

func newDecisionStore() DecisionStore {
	return DecisionStore{
		costs:   make(map[string]CostEstimate),
		history: make(map[CommandWithCondition]ResultHistory),
	}
}

type CostEstimate struct {
	Duration time.Duration
	Count    int

	// If false, we've only seen bootstrapped durations
	Real bool
}

// Creates a new cost estimate after working in the old cost estimate.
func (c CostEstimate) Add(log CommandLog, ctx LogContext) CostEstimate {
	isRealLog := ctx.Source != LogSourceBootstrap
	if isRealLog && !c.Real {
		// This is the first real log data
		return CostEstimate{Duration: log.Duration, Count: 1, Real: true}
	} else if c.Real && !isRealLog {
		// If we already have real logs, ignore the bootstrap log.
		return c
	}

	// Otherwise, fold in new data with a weighted average, so that
	// new data is worth at least 20%.
	oldCount := float64(c.Count)
	newCount := oldCount + 1
	oldWeight := oldCount/newCount - newCostExtraWeight
	newWeight := float64(1)/newCount + newCostExtraWeight
	newDuration := time.Duration(
		oldWeight*float64(c.Duration.Nanoseconds()) +
			newWeight*float64(log.Duration.Nanoseconds()))
	return CostEstimate{
		Duration: newDuration,
		Real:     c.Real,
		Count:    c.Count + 1,
	}
}

type WatCommandCostSort struct {
	Commands []WatCommand
	DS       DecisionStore
}

func (s WatCommandCostSort) Less(i, j int) bool {
	return s.DS.Cost(s.Commands[i]) < s.DS.Cost(s.Commands[j])
}

func (s WatCommandCostSort) Swap(i, j int) {
	s.Commands[i], s.Commands[j] = s.Commands[j], s.Commands[i]
}

func (s WatCommandCostSort) Len() int {
	return len(s.Commands)
}

type CommandWithCondition struct {
	Condition Condition
	Command   string
}

// The environment that a test is run in.
//
// Must be a value struct so that we can use it as a key in a map.
type Condition struct {
	// A known recently-edited file.
	EditedFile string

	// A known successful command.
	SuccessCommand string
}

func (c Condition) WithEditedFile(f string) Condition {
	c.EditedFile = f
	return c
}

func (c Condition) WithSuccess(cmd string) Condition {
	c.SuccessCommand = cmd
	return c
}

// Get all the conditions that are "ancestors" of this condition,
// from most narrow to most broad.
func (c Condition) Ancestors() []Condition {
	results := make([]Condition, 3)
	hasCommand := c.SuccessCommand != ""
	hasEditedFile := c.EditedFile != ""
	if hasCommand {
		results = append(results, c.WithSuccess(""))
	}
	if hasEditedFile {
		results = append(results, c.WithEditedFile(""))
	}
	if hasCommand && hasEditedFile {
		results = append(results, Condition{})
	}
	return results
}

type ResultHistory struct {
	SuccessCount uint32
	FailCount    uint32
}

func (h ResultHistory) Add(success bool) ResultHistory {
	successAdd := uint32(0)
	failAdd := uint32(0)
	if success {
		successAdd = 1
	} else {
		failAdd = 1
	}
	return ResultHistory{
		SuccessCount: h.SuccessCount + successAdd,
		FailCount:    h.FailCount + failAdd,
	}
}
