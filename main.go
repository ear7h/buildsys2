package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

//go:embed build-sys.dhall
var dhallFile string

//go:embed build-sys.sh
var scriptFile string

type Config struct {
	Actions []Action `json:"actions"`
	Dir     string   `json:"dir"`
}

type ActionType int

const (
	// Copy a file from Src to Dst. The paths must be relative.
	// One way to hack around this is to use ActionRun instead
	ActionCopy ActionType = iota
	// run the shell commands in Src and write the output
	// to Dst
	ActionRun
	// Set Src to an env var with name Dst
	ActionEnv
)

// An action
// TODO: change args to be []string, instead of multiple fields
type Action struct {
	// The action type
	Type ActionType `json:"type"`
	// an optional name
	Name string `json:"name"`
	// the first "argument"
	Src string `json:"src"`
	// the second "argument"
	Dst string `json:"dst"`
}

func (act *Action) DstEmpty() bool {
	return act.Dst == ""
}

func (act *Action) Execute(dest string) error {
	var err error
	dest, err = filepath.Abs(dest)
	if err != nil {
		return err
	}

	switch act.Type {
	case ActionCopy:
		if filepath.IsAbs(act.Src) {
			return fmt.Errorf("copy source not relative %s", act.Src)
		}

		if filepath.IsAbs(act.Dst) {
			return fmt.Errorf("copy destination not relative %s", act.Dst)
		}

		act.Src = fmt.Sprintf("cp %s %s", act.Src, filepath.Join(dest, act.Dst))
		act.Dst = ""
		fallthrough

	case ActionRun:
		fmt.Println(act.Src)

		cmd := exec.Command("bash", "-c", act.Src)
		cmd.Env = []string{"BUILDSYS_DIR=" + dest}
		if act.DstEmpty() {
			cmd.Stdout = os.Stdout
		} else {
			var err error
			cmd.Stdout, err = os.OpenFile(
				filepath.Join(dest, act.Dst),
				os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
				0644)
			if err != nil {
				return err
			}
		}

		return cmd.Run()

	case ActionEnv:
		return os.Setenv(act.Dst, act.Src)

	default:
		return fmt.Errorf("invalid action type %d", act.Type)
	}
}

var (
	flagConfig = flag.String("config", "-",
		"where to read the configuration file")
	flagNumber = flag.Bool("number", true,
		"append a '-' and a number when the output directory matches")
	flagDhall = flag.Bool("dhall", false,
		"print an example dhall file")
	flagScript = flag.Bool("script", false,
		"print an example script")
)

func exitError(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg, args...)
	os.Exit(1)
}

func main() {
	flag.Parse()

	if *flagDhall {
		_, err := io.Copy(os.Stdout, strings.NewReader(dhallFile))
		if err != nil {
			exitError("couldn't write to stdout: %v", err)
		}
		return
	}

	if *flagScript {
		_, err := io.Copy(os.Stdout, strings.NewReader(scriptFile))
		if err != nil {
			exitError("couldn't write to stdout: %v", err)
		}
		return
	}

	var configFile io.Reader
	if *flagConfig == "-" {
		configFile = os.Stdin
	} else if len(*flagConfig) == 0 {
		exitError("empty config\n")
	}

	var config Config

	err := json.NewDecoder(configFile).Decode(&config)
	if err != nil {
		exitError("could not parse config: %v\n", err)
	}

	dir := config.Dir
	err = os.Mkdir(dir, 0755)
	if err != nil {
		if !os.IsExist(err) || !*flagNumber {
			exitError("could not make output dir %s: %v\n", dir, err)
		}

		// IsExist error AND we can number it
		dirUp := filepath.Dir(dir)
		dirBase := filepath.Base(dir)
		patPrefix := dirBase + "-"
		matches, err := fs.Glob(os.DirFS(dirUp), patPrefix+"*")
		if err != nil {
			exitError("could not number directory: %v", err)
		}

		max := 1
		for _, v := range matches {
			n, _ := strconv.Atoi(v[len(patPrefix):])
			if n > max {
				max = n
			}
		}

		max++

		dir = dir + "-" + strconv.Itoa(max)
		err = os.Mkdir(dir, 0755)
		if err != nil {
			exitError("could not make numbered directory: %v", err)
		}
	}

	fmt.Println("writing to dir", dir)

	for i, v := range config.Actions {
		err = v.Execute(dir)
		if err != nil {
			var name interface{}

			if v.Name != "" {
				name = v.Name
			} else {
				name = i
			}

			exitError("error executing action %v: %v\n", name, err)
		}
	}
}
