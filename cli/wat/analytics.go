package wat

import (
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

	return a, c, nil
}
