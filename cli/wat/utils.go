package wat

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/pkg/term"
)

func Fatal(msg string, err error) {
	watlytics.errs.Write(err)
	fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
	os.Exit(1)
}

func UserYN(input rune, defaultVal bool) bool {
	if input == 13 || input == 0 { // carriage return or no input
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

	t, _ := term.Open("/dev/tty")
	term.RawMode(t)
	bytes := make([]byte, 1)

	_, err := t.Read(bytes)
	if err != nil {
		return 0, err
	}

	r := rune(bytes[0])

	t.Restore()
	t.Close()

	return r, nil
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
