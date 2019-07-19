package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

const Go = "go"

type Command struct {
	Name string
	Body func()
}

type BuildOptions struct {
	Env    map[string]string
	Stdout io.Writer
	Stderr io.Writer
}

type Build struct {
	verbose     bool
	env         map[string]string
	stdout      io.Writer
	stderr      io.Writer
	buildErrors []error

	currentTarget string

	cmds                map[string]func()
	cmdsRegistrationOrd []string
}

func NewBuild(o BuildOptions) *Build {
	result := &Build{
		env:    nil,
		stdout: os.Stdout,
		stderr: os.Stderr,

		cmds: make(map[string]func()),
	}
	if o.Env != nil {
		result.env = make(map[string]string, len(o.Env))
		for k, v := range o.Env {
			result.env[k] = v
		}
	}
	if o.Stdout != nil {
		result.stdout = o.Stdout
	}
	if o.Stderr != nil {
		result.stderr = o.Stderr
	}
	return result
}

func (b *Build) Run(cmd string, args ...string) {
	c := exec.Command(cmd, args...)
	c.Env = os.Environ()
	for k, v := range b.env {
		c.Env = append(c.Env, k+"="+v)
	}
	c.Stderr = b.stdout
	c.Stdout = b.stderr
	c.Stdin = os.Stdin
	if b.currentTarget != "" {
		fmt.Printf("[%s] ", b.currentTarget)
	}
	fmt.Printf("%s %s\n", cmd, strings.Join(args, " "))
	runErr := c.Run()
	if runErr != nil {
		b.buildErrors = append(b.buildErrors, runErr)
	}
}

func (b *Build) ForceRun(cmd string, args ...string) {
	b.Run(cmd, args...)
	b.buildErrors = nil
}

func (b *Build) RunCmd(cmd string, args ...string) func() {
	return func() {
		b.Run(cmd, args...)
	}
}

func (b *Build) RunForceCmd(cmd string, args ...string) func() {
	return func() {
		b.ForceRun(cmd, args...)
	}
}

func (b *Build) BashRun(cmd string, args ...string) {
	fullCmd := []string{cmd}
	fullCmd = append(fullCmd, args...)
	b.Run("bash", "-c", strings.Join(fullCmd, " "))
	b.buildErrors = nil
}

func (b *Build) Cmd(subCommand string, body func()) {
	_, ok := b.cmds[subCommand]
	if ok {
		b.buildErrors = append(
			b.buildErrors, fmt.Errorf("can't register command `%v`. Already has command with such name", subCommand),
		)
		return
	}
	if body == nil {
		b.buildErrors = append(
			b.buildErrors, fmt.Errorf("can't register command `%v`. Command body can't be nil", subCommand),
		)
		return
	}
	b.cmds[subCommand] = body
	b.cmdsRegistrationOrd = append(b.cmdsRegistrationOrd, subCommand)
}

func (b *Build) Build(args []string) {
	if len(b.buildErrors) > 0 {
		b.printAllErrorsAndExit()
		return
	}

	if len(args) == 0 {
		for _, cmd := range b.cmdsRegistrationOrd {
			b.currentTarget = cmd
			b.cmds[cmd]()
			if len(b.buildErrors) > 0 {
				b.printAllErrorsAndExit()
			}
		}
		return
	}

	if args[0] == "-h" {
		b.printAvailableTargets()
		return
	}

	for _, cmd := range args {
		if _, ok := b.cmds[cmd]; !ok {
			b.printAvailableTargets()
			fmt.Printf("can't find such command as: `%v`\n", cmd)
			fmt.Println("can't execute build")
			os.Exit(-1)
		}
	}
	for _, cmd := range args {
		b.currentTarget = cmd
		b.cmds[cmd]()
		if len(b.buildErrors) > 0 {
			b.printAllErrorsAndExit()
		}
	}
}

func (b *Build) printAvailableTargets() {
	fmt.Printf("available targets:\n")
	for _, cmd := range b.cmdsRegistrationOrd {
		fmt.Printf("	%+v\n", cmd)
	}
}

func (b *Build) printAllErrorsAndExit() {
	for _, err := range b.buildErrors {
		fmt.Printf("%v\n", err)
	}
	fmt.Println("can't execute build")
	os.Exit(-1)
}

func (b *Build) Register(commands []Command) {
	for _, cmd := range commands {
		b.Cmd(cmd.Name, cmd.Body)
	}
}

func main() {
	B.Register(Commands)
	B.Build(os.Args[1:])
}
