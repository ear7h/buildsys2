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
	"regexp"
	"strconv"
	"strings"
)

//go:embed build-sys.dhall
var dhallFile string

//go:embed build-sys.sh
var scriptFile string

type Config struct {
	Name     string   `json:"name"`
	Actions []Action `json:"actions"`
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

type ExecEnv struct {
	Dir, ParentDir string
}
func (env *ExecEnv) ToSlice() []string {
	return []string{
		"BUILDSYS_DIR=" + env.Dir,
		"BUILDSYS_PARENT_DIR=" + env.ParentDir,
	}
}

func (act *Action) Execute(env ExecEnv) error {

	switch act.Type {
	case ActionCopy:
		if filepath.IsAbs(act.Src) {
			return fmt.Errorf("copy source not relative %s", act.Src)
		}

		if filepath.IsAbs(act.Dst) {
			return fmt.Errorf("copy destination not relative %s", act.Dst)
		}

		act.Src = fmt.Sprintf("cp %s %s", act.Src, filepath.Join(env.Dir, act.Dst))
		act.Dst = ""
		fallthrough

	case ActionRun:
		fmt.Println(act.Src)

		cmd := exec.Command("bash", "-c", act.Src)
		cmd.Env = env.ToSlice()
		if act.DstEmpty() {
			cmd.Stdout = os.Stdout
		} else {
			var err error
			cmd.Stdout, err = os.OpenFile(
				filepath.Join(env.Dir, act.Dst),
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
	flagParentDir = flag.String("parent-dir", "output",
		"the parent directory of the build(s)")
	flagNumber = flag.Bool("number", true,
		"append a '-' and a number when the output directory matches")
	flagDhall = flag.Bool("dhall", false,
		"print an example dhall file")
	flagScript = flag.Bool("script", false,
		"print an example script")
	flagDry = flag.Bool("dry-run", false,
		"print actions but don't run")
)

var nameVerifyRegex = regexp.MustCompile("^[a-zA-Z\\-0-9]+$")

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

	var err error
	*flagParentDir, err = filepath.Abs(*flagParentDir)
	if err != nil {
		exitError("couldn't make parent-dir absolute: %v", err)
	}

	var configFile io.Reader
	if *flagConfig == "-" {
		configFile = os.Stdin
	} else if len(*flagConfig) == 0 {
		exitError("empty config\n")
	}

	var configs []Config

	err = json.NewDecoder(configFile).Decode(&configs)
	if err != nil {
		exitError("could not parse config: %v\n", err)
	}

	for _, config := range configs {
		if !nameVerifyRegex.MatchString(config.Name) {
			exitError("invalid name %q\n", config.Name)
		}

		for _, v := range flag.Args() {
			if v == config.Name {
				goto Ok
			}
		}
		continue

		Ok:

		env := ExecEnv {
			Dir : filepath.Join(*flagParentDir, config.Name),
			ParentDir : *flagParentDir,
		}
		fmt.Println("env: ", env)

		if *flagDry {
			for _, v := range config.Actions {
				fmt.Println(v)
			}
			continue
		}


		err = os.Mkdir(env.Dir, 0755)
		if err != nil {
			if !os.IsExist(err) || !*flagNumber {
				exitError("could not make output dir %s: %v\n", env.Dir, err)
			}

			// IsExist error AND we can number it
			patPrefix := config.Name + "-"
			matches, err := fs.Glob(os.DirFS(*flagParentDir), patPrefix+"*")
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

			env.Dir = env.Dir + "-" + strconv.Itoa(max)
			err = os.Mkdir(env.Dir, 0755)
			if err != nil {
				exitError("could not make numbered directory: %v", err)
			}
		}

		fmt.Println("writing to dir", env.Dir)

		for i, v := range config.Actions {
			err = v.Execute(env)
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
}
