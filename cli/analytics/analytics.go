package analytics

import (
	"os"
	"strings"

	"fmt"

	"github.com/spf13/cobra"
	"github.com/windmilleng/wat/cli/dirs"
)

const optInByDefault = false

func Init() (Analytics, *cobra.Command, error) {
	// TODO(maia): use this info in Analytics struct
	//optedIn := optedIn()
	//fmt.Fprintf(os.Stderr, "ANALYTICS OPTED-IN: %t\n", optedIn)

	a := NewMemoryAnalytics()
	c, err := initCLI()
	if err != nil {
		return nil, nil, err
	}

	return a, c, nil
}

type AnalyticsOpt int

const (
	AnalyticsOptDefault AnalyticsOpt = iota
	AnalyticsOptOut
	AnalyticsOptIn
)

var choices = map[AnalyticsOpt]string{
	AnalyticsOptDefault: "default",
	AnalyticsOptOut:     "opt-out",
	AnalyticsOptIn:      "opt-in",
}

func readChoiceFile() (string, error) {
	d, err := dirs.UseWindmillDir()
	if err != nil {
		return "", err
	}

	txt, err := d.ReadFile(choiceFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		txt = ""
	}

	return strings.TrimSpace(txt), nil
}

func optedIn() bool {
	txt, err := readChoiceFile()
	if txt != "" {
		switch txt {
		case choices[AnalyticsOptOut]:
			return false
		case choices[AnalyticsOptIn]:
			return true
		}
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "analytics.optedIn: %v\n", err)
	}
	return optInByDefault
}

// TODO(maia): all metrics add user hash as tag (or method: IncWithTagsAndUser?)
type Analytics interface {
	Count(name string, tags map[string]string, n int)
	// TODO(maia): default increment of 1
}

// Awkwardly just store stuff in memory for now
type MemoryAnalytics struct {
	Incs []IncEvent
}

type IncEvent struct {
	name string
	tags map[string]string
	n    int
}

func (a *MemoryAnalytics) Count(name string, tags map[string]string, n int) {
	fmt.Printf("%s += %d\n", name, n)
	a.Incs = append(a.Incs, IncEvent{name: name, tags: tags, n: n})
}

func NewMemoryAnalytics() *MemoryAnalytics {
	return &MemoryAnalytics{}
}

var _ Analytics = &MemoryAnalytics{}
