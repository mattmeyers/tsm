package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"slices"
	"strings"
)

var IgnoreDirs = []string{".git", ".vscode", ".idea"}

func main() {
	if err := run(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func run() error {
	config, err := readConfig()
	if err != nil {
		return err
	}

	targetDir, err := getTargetDir(config.BaseDirs)
	if err != nil {
		return err
	} else if targetDir == "" {
		return nil
	}

	id := path.Base(targetDir)

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

type Config struct {
	BaseDirs []string `json:"base_dirs"`
}

func readConfig() (Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return Config{}, err
	}

	f, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, err
	}

	var config Config
	err = json.Unmarshal(f, &config)
	if err != nil {
		return Config{}, err
	}

	return config, nil
}

func getConfigPath() (string, error) {
	configPath, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return path.Join(configPath, "tsm", "config.json"), nil
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

func getTargetDir(baseDirs []string) (string, error) {
	paths, err := listDirectories(baseDirs)
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

func listDirectories(baseDirs []string) ([]string, error) {
	var paths []string
	for _, baseDir := range baseDirs {
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

	return removeIgnoredDirs(paths), nil
}

func removeIgnoredDirs(paths []string) []string {
	return slices.DeleteFunc(paths, func(path string) bool {
		for _, d := range IgnoreDirs {
			if strings.HasSuffix(path, d) {
				return true
			}
		}

		return false
	})
}

func tmuxRunning() bool {
	err := runCommand(IO{}, "tmux", "ls")
	return err == nil
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
