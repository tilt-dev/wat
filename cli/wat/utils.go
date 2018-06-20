package wat

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/pkg/term"
)

const AsciiLineFeed = 10
const AsciiEnter = 13
const AsciiEsc = 27
const AsciiNone = 0

func UserYN(input rune, defaultVal bool) bool {
	if containsRune([]rune{AsciiLineFeed, AsciiEnter, AsciiNone}, input) {
		return defaultVal
	} else if input == 'y' || input == 'Y' {
		return true
	}
	return false
}

func TermBold(s string) string {
	return fmt.Sprintf("\033[1m%s\033[0m", s)
}

func MustJson(obj interface{}) string {
	b, err := json.Marshal(obj)
	if err != nil {
		panic(fmt.Sprintf("unexpected err %T: %+v, failed to jsonify: %v", obj, obj, err))
	}
	return string(b)
}

// getChar returns a single character entered by the user.
func getChar() (rune, error) {
	if !isatty.IsTerminal(os.Stdout.Fd()) {
		return 0, nil
	}

	t, err := term.Open("/dev/tty", term.CBreakMode)
	if err != nil {
		return 0, nil
	}
	defer func() {
		t.Restore()
		t.Close()
	}()

	bytes := make([]byte, 1)
	_, err = t.Read(bytes)
	if err != nil {
		return 0, err
	}

	return rune(bytes[0]), nil
}

// waitOnInterruptChar returns when:
// 1) the user types one of the interrupt chars, or
// 2) the context times out/cancels
// whichever comes first
func waitOnInterruptChar(ctx context.Context, interrupts []rune) error {
	if !isatty.IsTerminal(os.Stdout.Fd()) {
		return fmt.Errorf("No terminal available")
	}

	t, err := term.Open("/dev/tty", term.CBreakMode, term.ReadTimeout(200*time.Millisecond))
	if err != nil {
		return nil
	}
	defer func() {
		t.Restore()
		t.Close()
	}()

	// Keep polling until there is data on the terminal
	for true {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// We set a read timeout on the open file descriptor, so this will block until
		// we get data or there's a timeout.
		bytes := make([]byte, 1)
		n, _ := t.Read(bytes)
		if n == 0 {
			// We timed out! Try again on the next loop.
			continue
		}

		r := rune(bytes[0])
		if containsRune(interrupts, r) {
			return nil
		}
	}

	return nil
}

// dedupeAgainst returns elements of array a not in array b (i.e. the difference: A - B)
func dedupeAgainst(a, b []string) (res []string) {
	for _, elem := range a {
		if !contains(b, elem) {
			res = append(res, elem)
		}
	}
	return res
}

func contains(arr []string, s string) bool {
	for _, elem := range arr {
		if elem == s {
			return true
		}
	}
	return false
}

func containsRune(runes []rune, v rune) bool {
	for _, r := range runes {
		if v == r {
			return true
		}
	}
	return false
}
