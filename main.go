package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/rmf0112/CSCE4600/Project2/builtins"
	"github.com/shirou/gopsutil/mem"
)

var commandHelp = map[string]string{
	"cd":    "Change the current working directory",
	"env":   "Display or modify environment variables",
	"exit":  "Exit the shell",
	"ls":    "List files in the current directory",
	"pwd":   "Print the current working directory",
	"alloc": "Display information about memory allocation",
	"echo":  "Print arguments to the standard output",
	"help":  "Display information about available commands",
}

func main() {
	exit := make(chan struct{}, 2) // buffer this so there's no deadlock.
	runLoop(os.Stdin, os.Stdout, os.Stderr, exit)
}

func runLoop(r io.Reader, w, errW io.Writer, exit chan struct{}) {
	var (
		input    string
		err      error
		readLoop = bufio.NewReader(r)
	)
	for {
		select {
		case <-exit:
			_, _ = fmt.Fprintln(w, "exiting gracefully...")
			return
		default:
			if err := printPrompt(w); err != nil {
				_, _ = fmt.Fprintln(errW, err)
				continue
			}
			if input, err = readLoop.ReadString('\n'); err != nil {
				_, _ = fmt.Fprintln(errW, err)
				continue
			}
			if err = handleInput(w, input, exit); err != nil {
				_, _ = fmt.Fprintln(errW, err)
			}
		}
	}
}

func printPrompt(w io.Writer) error {
	// Get current user.
	// Don't prematurely memoize this because it might change due to `su`?
	u, err := user.Current()
	if err != nil {
		return err
	}
	// Get current working directory.
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	// /home/User [Username] $
	_, err = fmt.Fprintf(w, "%v [%v] $ ", wd, u.Username)

	return err
}

func handleInput(w io.Writer, input string, exit chan<- struct{}) error {
	// Remove trailing spaces.
	input = strings.TrimSpace(input)

	// Split the input separate the command name and the command arguments.
	args := strings.Split(input, " ")
	name, args := args[0], args[1:]

	// Check for built-in commands.
	// New builtin commands should be added here. Eventually this should be refactored to its own func.
	switch name {
	case "cd":
		return builtins.ChangeDirectory(args...)
	case "env":
		return builtins.EnvironmentVariables(w, args...)
	case "exit":
		exit <- struct{}{}
		return nil
	case "ls":
		return listFiles(w, args...)
	case "pwd":
		return printWorkingDirectory(w)
	case "alloc":
		return allocateMemory(w, args...)
	case "echo":
		return echo(w, args...)
	case "help":
		return showHelp(w, args...)
	}

	return executeCommand(name, args...)
}

func listFiles(w io.Writer, args ...string) error {
	dir, err := os.Open(".")
	if err != nil {
		return err
	}
	defer dir.Close()

	fileInfos, err := dir.Readdir(-1)
	if err != nil {
		return err
	}

	for _, fileInfo := range fileInfos {
		_, _ = fmt.Fprintf(w, "%s\t", fileInfo.Name())
	}

	_, _ = fmt.Fprintln(w)

	return nil
}

func printWorkingDirectory(w io.Writer) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(w, wd)

	return err
}

func allocateMemory(w io.Writer, args ...string) error {
	if len(args) != 0 {
		return fmt.Errorf("Usage: Alloc")
	}

	memoryInfo, err := mem.VirtualMemory()
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "Total: %v, Free: %v\n", memoryInfo.Total, memoryInfo.Free)
	return err
}

func echo(w io.Writer, args ...string) error {
	_, err := fmt.Fprintln(w, strings.Join(args, " "))
	return err
}

func showHelp(w io.Writer, args ...string) error {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(w, "Available commands:")
		for cmd, desc := range commandHelp {
			_, _ = fmt.Fprintf(w, "  %s: %s\n", cmd, desc)
		}
		return nil
	}

	cmd := args[0]
	if desc, ok := commandHelp[cmd]; ok {
		_, _ = fmt.Fprintf(w, "Help for %s:\n%s\n", cmd, desc)
	} else {
		_, _ = fmt.Fprintf(w, "Unknown command: %s\n", cmd)
	}

	return nil
}

func executeCommand(name string, arg ...string) error {
	// Otherwise prep the command
	cmd := exec.Command(name, arg...)

	// Set the correct output device.
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	// Execute the command and return the error.
	return cmd.Run()
}
