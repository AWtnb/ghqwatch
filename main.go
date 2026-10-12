package main

import (
	_ "embed"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"fyne.io/systray"
	"github.com/gen2brain/beeep"
)

//go:embed icon.ico
var icon []byte

const (
	interval   = 1 * time.Hour
	retryAfter = 3 * time.Minute
	workers    = 4
)

var errAllFailed = errors.New("fetch failed for all repositories (network unavailable?)")

func run(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	// Hide console window
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func listRepos() ([]string, error) {
	out, err := run("", "ghq", "list", "-p")
	if err != nil {
		return nil, err
	}
	var repos []string
	for _, l := range strings.Split(out, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			repos = append(repos, l)
		}
	}
	return repos, nil
}

func isBehind(path string) (bool, error) {
	if _, err := run(path, "git", "fetch", "--quiet"); err != nil {
		return false, err
	}
	out, err := run(path, "git", "rev-list", "--count", "HEAD..@{u}")
	if err != nil {
		return false, nil // upstream 無し、detached HEAD など
	}
	return out != "0", nil
}

type result struct {
	behinds []string
	failed  int
	total   int
}

func checkAll(onProgress func(done, total int)) (result, error) {
	repos, err := listRepos()
	if err != nil {
		return result{}, err
	}
	var (
		mu      sync.Mutex
		behinds []string
		failed  int
		done    int
		wg      sync.WaitGroup
		ch      = make(chan string)
	)
	for range workers {
		wg.Go(func() {
			for p := range ch {
				b, err := isBehind(p)
				mu.Lock()
				switch {
				case err != nil:
					failed++
				case b:
					behinds = append(behinds, p)
				}
				done++
				d := done
				mu.Unlock()
				onProgress(d, len(repos))
			}
		})
	}
	for _, r := range repos {
		ch <- r
	}
	close(ch)
	wg.Wait()
	sort.Strings(behinds)

	res := result{behinds: behinds, failed: failed, total: len(repos)}
	if 0 < len(repos) && failed == len(repos) {
		return res, errAllFailed
	}
	return res, nil
}

func updateAll(paths []string) (ok int) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, workers)
	for _, p := range paths {
		wg.Add(1)
		sem <- struct{}{}
		go func(p string) {
			defer wg.Done()
			defer func() { <-sem }()
			if _, err := run(p, "git", "pull", "--ff-only", "--quiet"); err == nil {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}(p)
	}
	wg.Wait()
	return
}

func main() { systray.Run(onReady, nil) }

func onReady() {
	systray.SetIcon(icon)
	systray.SetTitle("ghqwatch")
	systray.SetTooltip("ghqwatch")

	mStatus := systray.AddMenuItem("Unchecked", "")
	mStatus.Disable()
	systray.AddSeparator()
	mCheck := systray.AddMenuItem("Check now", "")
	mUpdate := systray.AddMenuItem("Update", "")
	mUpdate.Hide()
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "")

	var (
		busy    sync.Mutex
		behinds []string
		lastKey string
	)

	check := func() bool {
		if !busy.TryLock() {
			return false
		}
		defer busy.Unlock()

		mCheck.Disable()
		defer mCheck.Enable()

		res, err := checkAll(func(d, t int) {
			mStatus.SetTitle(fmt.Sprintf("Checking (%d/%d)", d, t))
		})
		now := time.Now().Format("15:04")
		if err != nil {
			mStatus.SetTitle(fmt.Sprintf("ERROR (%s): %s", now, err))
			systray.SetTooltip("ghqwatch: check failed")
			return true
		}

		behinds = res.behinds
		suffix := ""
		if 0 < res.failed {
			suffix = fmt.Sprintf(", %d failed", res.failed)
		}
		if len(res.behinds) == 0 {
			mStatus.SetTitle(fmt.Sprintf("Up-to-date (%s%s)", now, suffix))
			mUpdate.Hide()
			systray.SetTooltip("ghqwatch: up-to-date" + suffix)
			lastKey = ""
			return 0 < res.failed
		}
		mStatus.SetTitle(fmt.Sprintf("%d updates available (%s%s)", len(res.behinds), now, suffix))
		mUpdate.SetTitle(fmt.Sprintf("Update %d", len(res.behinds)))
		mUpdate.Show()
		systray.SetTooltip(fmt.Sprintf("ghqwatch: %d updates available", len(res.behinds)))

		if key := strings.Join(res.behinds, "|"); key != lastKey {
			lastKey = key
			_ = beeep.Notify("ghqwatch", fmt.Sprintf("%d repositories have updates available", len(res.behinds)), "")
		}
		return 0 < res.failed
	}

	update := func() {
		if !busy.TryLock() {
			return
		}
		targets := behinds
		mStatus.SetTitle("Updating...")
		ok := updateAll(targets)
		busy.Unlock()
		_ = beeep.Notify("ghqwatch", fmt.Sprintf("Updated %d/%d repositories", ok, len(targets)), "")
		check()
	}

	// If a retry is needed, retry after retryAfter and repeat until successful.
	var retryTimer *time.Timer
	var retryMu sync.Mutex
	var runCheck func()
	runCheck = func() {
		need := check()
		retryMu.Lock()
		defer retryMu.Unlock()
		if retryTimer != nil {
			retryTimer.Stop()
			retryTimer = nil
		}
		if need {
			retryTimer = time.AfterFunc(retryAfter, runCheck)
		}
	}

	go runCheck()
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				go runCheck()
			case <-mCheck.ClickedCh:
				go runCheck()
			case <-mUpdate.ClickedCh:
				go update()
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}
