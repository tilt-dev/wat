package wat

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/windmilleng/wat/cli/analytics"
)

// Stat names
const (
	statFatal               = "fatal"
	statInit                = "init"
	statRecommendation      = "recommendation"
	statTrainingInterrupted = "training_interrupted"

	timerCommandsRun = "commands_run"
	timerDecide      = "decide"
)

// Tags for stats
const (
	tagAccepted = "accepted"
	tagDir      = "dir"
	tagError    = "error"
)

func initAnalytics() (analytics.Analytics, *cobra.Command, error) {
	a, c, err := analytics.Init(appNameWat)
	if err != nil {
		return nil, nil, err
	}

	// Nudge the user if default
	o, err := analytics.OptStatus()
	if err != nil {
		return nil, nil, err
	}

	if o == analytics.OptDefault {
		// NB(dbentley): we could ask them to pick here, but it would slow down the initial experience
		fmt.Println("[ psst! help us learn with telemetry data; run \"wat analytics opt in\" (or \"out\") to stop seeing this message ]")
	}

	return a, c, nil
}
