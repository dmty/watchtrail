package tlsca

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

const linuxCAPath = "/usr/local/share/ca-certificates/watchtrail-ca.crt"

// trustPlan is a platform-neutral description of a trust-store change.
type trustPlan struct {
	CopyTo     string   // copy caPath here before running (install)
	RemoveFile string   // delete this file (uninstall)
	Run        []string // command + args to execute
	NeedsSudo  bool     // Run requires root (Linux); macOS prompts via GUI instead
}

func installPlan(goos, caPath string) (trustPlan, error) {
	switch goos {
	case "darwin":
		// User login keychain (no -d/-k): `security` shows a GUI auth prompt,
		// no sudo needed, and the trust is honored for the current user.
		return trustPlan{Run: []string{
			"security", "add-trusted-cert", "-r", "trustRoot", caPath,
		}}, nil
	case "linux":
		return trustPlan{
			CopyTo:    linuxCAPath,
			Run:       []string{"update-ca-certificates"},
			NeedsSudo: true,
		}, nil
	default:
		return trustPlan{}, fmt.Errorf("trust install not automated on %s; trust %s manually", goos, caPath)
	}
}

func uninstallPlan(goos, caPath string) (trustPlan, error) {
	switch goos {
	case "darwin":
		return trustPlan{Run: []string{"security", "remove-trusted-cert", caPath}}, nil
	case "linux":
		return trustPlan{
			RemoveFile: linuxCAPath,
			Run:        []string{"update-ca-certificates", "--fresh"},
			NeedsSudo:  true,
		}, nil
	default:
		return trustPlan{}, fmt.Errorf("trust removal not automated on %s; remove %s manually", goos, caPath)
	}
}

// Install plants the CA in the host OS trust store. On macOS the `security`
// tool raises its own GUI auth prompt (login keychain, no sudo); on Linux the
// copy + update needs root.
func Install(caPath string) error {
	plan, err := installPlan(runtime.GOOS, caPath)
	if err != nil {
		return err
	}
	if plan.CopyTo != "" {
		data, err := os.ReadFile(caPath)
		if err != nil {
			return err
		}
		if err := os.WriteFile(plan.CopyTo, data, 0o644); err != nil {
			return fmt.Errorf("write %s (re-run with sudo): %w", plan.CopyTo, err)
		}
	}
	cmd := exec.Command(plan.Run[0], plan.Run[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

// UninstallCommand returns a copy-pasteable command that removes the CA from
// the trust store. disable-tls prints this rather than running it, so the user
// isn't forced through a second privilege prompt.
func UninstallCommand(caPath string) (string, error) {
	plan, err := uninstallPlan(runtime.GOOS, caPath)
	if err != nil {
		return "", err
	}
	sudo := ""
	if plan.NeedsSudo {
		sudo = "sudo "
	}
	var parts []string
	if plan.RemoveFile != "" {
		parts = append(parts, sudo+"rm "+plan.RemoveFile)
	}
	parts = append(parts, sudo+strings.Join(plan.Run, " "))
	return strings.Join(parts, " && "), nil
}
