package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/gen2brain/beeep"
)

const maxNotifyNames = 3

func joinFileNames(behinds []string, sep string) string {
	names := make([]string, 0, len(behinds))
	for _, p := range behinds {
		names = append(names, filepath.Base(p))
	}
	if len(names) <= maxNotifyNames {
		return strings.Join(names, sep)
	}
	return strings.Join(names[:maxNotifyNames], sep) + fmt.Sprintf(" (+%d)", len(names)-maxNotifyNames)
}

func main() {
	n := flag.Int("n", 10, "Number of dummy repositories")
	long := flag.Bool("long", false, "Use long names")
	flag.Parse()

	var behinds []string
	for i := 1; i <= *n; i++ {
		name := fmt.Sprintf("repo-%02d", i)
		if *long {
			name = fmt.Sprintf("very-long-repository-name-for-testing-%02d", i)
		}
		behinds = append(behinds, filepath.Join(`C:\ghq\github.com\user`, name))
	}

	err := beeep.Notify(
		"ghqwatch",
		fmt.Sprintf("%d repositories have updates available:\n%s", len(behinds), joinFileNames(behinds, "\n")),
		"",
	)
	if err != nil {
		fmt.Println("notify error:", err)
	}
	time.Sleep(time.Second) // Keep the process alive until the notification is sent
}
