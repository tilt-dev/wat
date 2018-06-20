package wat

import (
	"github.com/spf13/cobra"
	"github.com/windmilleng/wat/cli/analytics"
)

// Stat names
const (
	statFatal          string = "fatal"
	statInit           string = "init"
	statRecommendation string = "recommendation"
)

// Tags for stats
const (
	tagAccepted string = "accepted"
	tagDir      string = "dir"
	tagError    string = "error"
)

func initAnalytics() (analytics.Analytics, *cobra.Command, error) {
	a, c, err := analytics.Init()
	if err != nil {
		return nil, nil, err
	}

	return a, c, nil
}
