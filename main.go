package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"slices"
	"strings"
)

const AppUsage = `tsm - The Tmux Session Manager

tsm manages your tmux sessions by creating a new session per project directory.
Sessions may contain multiple windows which are isolated and maintained when
switching between projects. Omitting any commands will trigger the session
switcher.

USAGE:
    tsm [OPTIONS] [COMMAND]

COMMANDS:
    0                     Switch to the zero session.

OPTIONS:
    -h, --help            Show this help message.
`

func main() {
	if err := run(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func run() error {
	flag.Usage = func() { fmt.Print(AppUsage) }
	flag.Parse()

	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	config, err := readConfig(configPath)
	if err != nil {
		return err
	}

	switch flag.Arg(0) {
	case "0":
		return handleSwitchToZero()
	default:
		return handleSessionSwitch(config)
	}
}

type Config struct {
	BaseDirs   []string `json:"base_dirs"`
	IgnoreDirs []string `json:"ignore_dirs"`
}

func getConfigPath() (string, error) {
	configPath, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return path.Join(configPath, "tsm", "config.json"), nil
}

func readConfig(configPath string) (Config, error) {
	f, err := os.ReadFile(configPath)
	if errors.Is(err, os.ErrNotExist) {
		c := Config{BaseDirs: []string{}, IgnoreDirs: []string{}}
		return c, writeConfig(configPath, c)
	} else if err != nil {
		return Config{}, err
	}

	var config Config
	err = json.Unmarshal(f, &config)
	if err != nil {
		return Config{}, err
	}

	return config, nil
}

func writeConfig(configPath string, config Config) error {
	d, err := json.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, d, 0644)
}

func handleSessionSwitch(config Config) error {
	targetDir, err := getTargetDir(config)
	if err != nil {
		return err
	} else if targetDir == "" {
		return nil
	}

	id := cleanID(path.Base(targetDir))

	if !sessionExists(id) {
		err = createSession(id, targetDir)
		if err != nil {
			return err
		}
	}

	err = switchToSession(id)
	if err != nil {
		return err
	}

	return nil
}

func handleSwitchToZero() error {
	id := "0"
	targetDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if !sessionExists(id) {
		err = createSession(id, targetDir)
		if err != nil {
			return err
		}
	}

	err = switchToSession(id)
	if err != nil {
		return err
	}

	return nil

}

type IO struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

var stdIO = IO{
	Stdin:  os.Stdin,
	Stdout: os.Stdout,
	Stderr: os.Stderr,
}

func characterAllowed(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '-' ||
		r == '_'
}
func cleanID(id string) string {
	idSlice := []rune(id)
	for i, r := range idSlice {
		if !characterAllowed(r) {
			idSlice[i] = '_'
		}
	}

	return string(idSlice)
}

func getTargetDir(config Config) (string, error) {
	paths, err := listDirectories(config)
	if err != nil {
		return "", err
	}

	out := bytes.NewBuffer([]byte{})
	err = runCommand(IO{
		Stdin:  strings.NewReader(strings.Join(paths, "\n")),
		Stdout: out,
		Stderr: os.Stderr,
	}, "fzf")
	if err != nil {
		return "", nil
	}

	return strings.TrimSpace(out.String()), nil
}

func listDirectories(config Config) ([]string, error) {
	var paths []string
	for _, baseDir := range config.BaseDirs {
		d, err := os.ReadDir(baseDir)
		if err != nil {
			return nil, err
		}

		for _, entry := range d {
			if !entry.IsDir() {
				continue
			}

			paths = append(paths, path.Join(baseDir, entry.Name()))
		}
	}

	return removeIgnoredDirs(paths, config), nil
}

func removeIgnoredDirs(paths []string, config Config) []string {
	return slices.DeleteFunc(paths, func(path string) bool {
		for _, d := range config.IgnoreDirs {
			if strings.HasSuffix(path, d) {
				return true
			}
		}

		return false
	})
}

func sessionExists(id string) bool {
	err := runCommand(IO{}, "tmux", "has-session", "-t", id)
	return err == nil
}

func createSession(id, targetDir string) error {
	return runCommand(IO{}, "tmux", "new-session", "-d", "-s", id, "-c", targetDir)
}

func switchToSession(id string) error {
	if _, ok := os.LookupEnv("TMUX"); ok {
		return switchSession(id)
	}

	return attachToSession(id)
}

func attachToSession(id string) error {
	return runCommand(stdIO, "tmux", "attach", "-t", id)
}

func switchSession(id string) error {
	return runCommand(stdIO, "tmux", "switch-client", "-t", id)
}

func runCommand(inOut IO, command ...string) error {
	if len(command) == 0 {
		panic("tsm: empty command provided")
	}

	cmd := exec.Command(command[0], command[1:]...)

	cmd.Stdin = inOut.Stdin
	cmd.Stdout = inOut.Stdout
	cmd.Stderr = inOut.Stderr

	err := cmd.Run()

	return err
}
