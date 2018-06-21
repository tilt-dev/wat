package wat

import (
	"math"
	"reflect"
	"testing"
	"time"
)

var fileA = fileInfo{name: "a.txt", modTime: time.Now()}
var fileB = fileInfo{name: "b.txt", modTime: time.Now().Add(-time.Minute)}
var fileC = fileInfo{name: "c.txt", modTime: time.Now().Add(-time.Minute * 2)}
var fileD = fileInfo{name: "d.txt", modTime: time.Now().Add(-time.Minute * 3)}

var cmdA = WatCommand{Command: "cat a.txt", FilePattern: "a.txt"}
var cmdB = WatCommand{Command: "cat b.txt", FilePattern: "b.txt"}
var cmdC = WatCommand{Command: "cat c.txt", FilePattern: "c.txt"}
var cmdD = WatCommand{Command: "cat d.txt", FilePattern: "d.txt"}

var cmdLogA = newTestLog(cmdA, time.Minute, true)
var cmdLogB = newTestLog(cmdB, 30*time.Second, true)
var cmdLogC = newTestLog(cmdC, 20*time.Second, true)
var cmdLogD = newTestLog(cmdD, 10*time.Second, true)

var cmdLogSecSuccessA = newTestLog(cmdA, time.Second, true)
var cmdLogSecSuccessB = newTestLog(cmdB, time.Second, true)
var cmdLogSecSuccessC = newTestLog(cmdC, time.Second, true)
var cmdLogSecSuccessD = newTestLog(cmdD, time.Second, true)
var cmdLogSecFailA = newTestLog(cmdA, time.Second, false)
var cmdLogSecFailB = newTestLog(cmdB, time.Second, false)
var cmdLogSecFailC = newTestLog(cmdC, time.Second, false)
var cmdLogSecFailD = newTestLog(cmdD, time.Second, false)
var cmdLogMinuteFailA = newTestLog(cmdA, time.Minute, false)

func TestNoCommands(t *testing.T) {
	results := decideWith(nil, nil, []fileInfo{fileA, fileB, fileC, fileD}, 10)
	if len(results) != 0 {
		t.Fatalf("Expected 0 results. Actual: %d", len(results))
	}
}

func TestThreeCommands(t *testing.T) {
	results := decideWith([]WatCommand{cmdA, cmdB, cmdC}, nil, []fileInfo{fileA, fileB, fileC, fileD}, 3)
	expected := []WatCommand{cmdA, cmdB, cmdC}
	if !reflect.DeepEqual(results, expected) {
		t.Fatalf("Expected %+v. Actual: %+v", expected, results)
	}
}

func TestThreeCommandsToOne(t *testing.T) {
	results := decideWith([]WatCommand{cmdD, cmdB, cmdC}, nil, []fileInfo{fileA, fileB, fileC, fileD}, 1)
	expected := []WatCommand{cmdB}
	if !reflect.DeepEqual(results, expected) {
		t.Fatalf("Expected %+v. Actual: %+v", expected, results)
	}
}

func TestThreeCommandsOutOfOrder(t *testing.T) {
	results := decideWith([]WatCommand{cmdC, cmdA, cmdB}, nil, []fileInfo{fileA, fileB, fileC, fileD}, 3)
	expected := []WatCommand{cmdA, cmdB, cmdC}
	if !reflect.DeepEqual(results, expected) {
		t.Fatalf("Expected %+v. Actual: %+v", expected, results)
	}
}

func TestMissingFiles(t *testing.T) {
	results := decideWith([]WatCommand{cmdA, cmdB, cmdC}, nil, []fileInfo{fileB}, 3)
	expected := []WatCommand{cmdB, cmdA, cmdC}
	if !reflect.DeepEqual(results, expected) {
		t.Fatalf("Expected %+v. Actual: %+v", expected, results)
	}
}

func TestCost(t *testing.T) {
	ds := newDecisionStore()
	if ds.HasCost(cmdA) {
		t.Fatalf("Expected HasCost: false")
	}

	ds.addCommandCost(CommandLog{Command: cmdA.Command, Duration: time.Minute}, LogContext{})
	if !ds.HasCost(cmdA) {
		t.Fatalf("Expected HasCost: true")
	}

	if ds.Cost(cmdA) != time.Minute {
		t.Errorf("Expected cost %s. Actual %s", time.Minute, ds.Cost(cmdA))
	}

	ds.addCommandCost(CommandLog{Command: cmdA.Command, Duration: time.Second}, LogContext{})
	expected := time.Duration(0.3*float64(time.Minute.Nanoseconds()) + 0.7*float64(time.Second.Nanoseconds()))
	if ds.Cost(cmdA) != expected {
		t.Errorf("Expected cost %s. Actual %s", expected, ds.Cost(cmdA))
	}
}

func TestFailureDecide(t *testing.T) {
	group := CommandLogGroup{
		Logs: []CommandLog{
			cmdLogSecFailA,
			cmdLogSecFailA,
			cmdLogSecSuccessB,
			cmdLogSecFailB,
			cmdLogSecFailC,
			cmdLogSecFailC,
			cmdLogSecSuccessD,
			cmdLogSecFailD,
		},
		Context: LogContext{StartTime: time.Now(), Source: LogSourceUser},
	}
	results := decideWith([]WatCommand{cmdA, cmdB, cmdC, cmdD}, []CommandLogGroup{group}, nil, 3)
	expected := []WatCommand{cmdA, cmdC, cmdB}
	if !reflect.DeepEqual(results, expected) {
		t.Fatalf("Expected %+v. Actual: %+v", expected, results)
	}
}

func TestCostSensitiveGainDecide(t *testing.T) {
	group := CommandLogGroup{
		Logs: []CommandLog{
			// Because cmdA takes a minute, even though it fails all the time,
			// it becomes the worst choice.
			cmdLogMinuteFailA,
			cmdLogMinuteFailA,
			cmdLogSecSuccessB,
			cmdLogSecFailB,
			cmdLogSecFailC,
			cmdLogSecFailC,
			cmdLogSecSuccessD,
			cmdLogSecFailD,
		},
		Context: LogContext{StartTime: time.Now(), Source: LogSourceUser},
	}
	results := decideWith([]WatCommand{cmdA, cmdB, cmdC, cmdD}, []CommandLogGroup{group}, nil, 3)
	expected := []WatCommand{cmdC, cmdB, cmdD}
	if !reflect.DeepEqual(results, expected) {
		t.Fatalf("Expected %+v. Actual: %+v", expected, results)
	}
}

func TestCorrelationSensitiveGainDecide(t *testing.T) {
	// Create groups where A + B are highly correlated,
	// so we will not choose B if A is the first task.
	group1 := CommandLogGroup{
		Logs: []CommandLog{
			cmdLogSecSuccessA,
			cmdLogSecSuccessB,
			cmdLogSecSuccessC,
		},
		Context: LogContext{StartTime: time.Now(), Source: LogSourceUser},
	}
	group2 := CommandLogGroup{
		Logs: []CommandLog{
			cmdLogSecSuccessA,
			cmdLogSecSuccessB,
			cmdLogSecFailC,
		},
		Context: LogContext{StartTime: time.Now(), Source: LogSourceUser},
	}
	group3 := CommandLogGroup{
		Logs: []CommandLog{
			cmdLogSecFailA,
			cmdLogSecFailB,
		},
		Context: LogContext{StartTime: time.Now(), Source: LogSourceUser},
	}

	results := decideWith([]WatCommand{cmdA, cmdC, cmdB},
		[]CommandLogGroup{group1, group2, group3, group3, group3}, nil, 3)
	expected := []WatCommand{cmdA, cmdC, cmdB}
	if !reflect.DeepEqual(results, expected) {
		t.Fatalf("Expected %+v. Actual: %+v", expected, results)
	}
}

func TestFailureProbabilityDifferentPackage(t *testing.T) {
	ds := newDecisionStore()

	condB := Condition{EditedFile: "b.txt"}
	prob := ds.FailureProbability(cmdA, condB)
	expected := 0.5
	if !roughlyEqual(prob, expected) {
		t.Fatalf("Expected %v, actual: %v", expected, prob)
	}

	ctx := LogContext{}
	ds.addCommandHistory(cmdLogA, ctx, Condition{})
	prob = ds.FailureProbability(cmdA, condB)
	expected = failProbabilityZeroCase / (1 + failProbabilityZeroCase)
	if !roughlyEqual(prob, expected) {
		t.Fatalf("Expected %v, actual: %v", expected, prob)
	}

	ds.addCommandHistory(cmdLogSecFailA, ctx, Condition{})
	prob = ds.FailureProbability(cmdA, condB)
	expected = 0.5
	if !roughlyEqual(prob, expected) {
		t.Fatalf("Expected %v, actual: %v", expected, prob)
	}

	ds.addCommandHistory(cmdLogSecFailA, ctx, Condition{})
	prob = ds.FailureProbability(cmdA, condB)
	expected = 0.666
	if !roughlyEqual(prob, expected) {
		t.Fatalf("Expected %v, actual: %v", expected, prob)
	}

	// Logs that are sensitive to the most recently edited file
	// make the narrower probability kick in.
	ds.addCommandHistory(cmdLogA, LogContext{RecentEdits: []string{"b.txt"}}, Condition{})
	prob = ds.FailureProbability(cmdA, condB)
	expected = failProbabilityZeroCase / (1 + failProbabilityZeroCase)
	if !roughlyEqual(prob, expected) {
		t.Fatalf("Expected %v, actual: %v", expected, prob)
	}
}

func TestFailureProbabilitySamePackage(t *testing.T) {
	ds := newDecisionStore()

	condA := Condition{EditedFile: "a.txt"}
	prob := ds.FailureProbability(cmdA, condA)
	expected := 0.5
	if !roughlyEqual(prob, expected) {
		t.Fatalf("Expected %v, actual: %v", expected, prob)
	}

	ctx := LogContext{}
	ds.addCommandHistory(cmdLogA, ctx, Condition{})
	prob = ds.FailureProbability(cmdA, condA)
	expected = 0.5
	if !roughlyEqual(prob, expected) {
		t.Fatalf("Expected %v, actual: %v", expected, prob)
	}

	ds.addCommandHistory(cmdLogSecFailA, ctx, Condition{})
	prob = ds.FailureProbability(cmdA, condA)
	expected = 0.5
	if !roughlyEqual(prob, expected) {
		t.Fatalf("Expected %v, actual: %v", expected, prob)
	}

	ds.addCommandHistory(cmdLogSecFailA, ctx, Condition{})
	prob = ds.FailureProbability(cmdA, condA)
	expected = 0.666
	if !roughlyEqual(prob, expected) {
		t.Fatalf("Expected %v, actual: %v", expected, prob)
	}
}

func TestCostDecide(t *testing.T) {
	// Create log data that says D is faster than B, and only has data for B+D
	ds := newDecisionStore()
	ds.addCommandCost(cmdLogB, LogContext{})
	ds.addCommandCost(cmdLogD, LogContext{})
	results := secondTierDecideWith([]WatCommand{cmdA, cmdB, cmdC, cmdD}, ds, nil, 3)
	expected := []WatCommand{cmdD, cmdB, cmdA}
	if !reflect.DeepEqual(results, expected) {
		t.Fatalf("Expected %+v. Actual: %+v", expected, results)
	}
}

func roughlyEqual(a, b float64) bool {
	diff := math.Abs(a - b)
	return diff < 0.001
}

func newTestLog(cmd WatCommand, dur time.Duration, success bool) CommandLog {
	return CommandLog{
		Command:  cmd.Command,
		Success:  success,
		Duration: dur,
	}
}
