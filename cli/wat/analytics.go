package wat

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/windmilleng/wat/cli/analytics"
)

func initAnalytics() (analytics.Analytics, *cobra.Command, error) {
	a, c, err := analytics.Init("wat")
	if err != nil {
		return nil, nil, err
	}

	watlytics = &watAnalytics{}

	w, err := a.Register("init", analytics.Nil)
	if err != nil {
		return nil, nil, err
	}
	watlytics.init = analytics.NewStringWriter(w)

	w, err = a.Register("recs", analytics.Nil)
	if err != nil {
		return nil, nil, err
	}
	watlytics.recs = &recEventWriter{del: w}

	w, err = a.Register("errors", analytics.Nil)
	if err != nil {
		return nil, nil, err
	}
	watlytics.errs = analytics.NewErrorWriter(w)

	return a, c, nil
}

type watAnalytics struct {
	init analytics.StringWriter
	recs *recEventWriter
	errs analytics.ErrorWriter
}

var watlytics *watAnalytics

type recEvent struct {
	accepted    bool
	userLatency time.Duration
	runLatency  time.Duration
	dataAvail   int
}

type recEventWriter struct {
	del analytics.AnyWriter
}

func (w *recEventWriter) Write(e recEvent) {
	w.del.Write(e)
}
