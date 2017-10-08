package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os/exec"
	"regexp"
	"strings"

	"github.com/robfig/cron"
)

func call(stack []*exec.Cmd, pipes []*io.PipeWriter) (err error) {
	if stack[0].Process == nil {
		if err = stack[0].Start(); err != nil {
			return err
		}
	}
	if len(stack) > 1 {
		if err = stack[1].Start(); err != nil {
			return err
		}
		defer func() {
			if err == nil {
				pipes[0].Close()
				err = call(stack[1:], pipes[1:])
			}
		}()
	}
	return stack[0].Wait()
}

func Execute(output_buffer *bytes.Buffer, stack ...*exec.Cmd) (err error) {
	var error_buffer bytes.Buffer
	pipe_stack := make([]*io.PipeWriter, len(stack)-1)
	i := 0
	for ; i < len(stack)-1; i++ {
		stdin_pipe, stdout_pipe := io.Pipe()
		stack[i].Stdout = stdout_pipe
		stack[i].Stderr = &error_buffer
		stack[i+1].Stdin = stdin_pipe
		pipe_stack[i] = stdout_pipe
	}
	stack[i].Stdout = output_buffer
	stack[i].Stderr = &error_buffer

	if err := call(stack, pipe_stack); err != nil {
		log.Fatalln(string(error_buffer.Bytes()), err)
	}
	return err
}

//Function that returns the current wlan0 address as a string
func getIP() string {
	var b bytes.Buffer
	var str string
	if err := Execute(&b,
		//Since piping commands are a bit of a pain, using the above functions call() and Execute(), execute "/sbin/ifconfig wlan0 | grep 'inet addr:' | cut -d -f2 | awk '{print $1}'"
		exec.Command("/sbin/ifconfig", "wlan0"),
		exec.Command("grep", "inet addr:"),
		exec.Command("cut", "-d:", "-f2"),
		exec.Command("awk", "{print $1}"),
	); err != nil {
		log.Fatalln(err)
	}
	str = b.String()
	regex, err := regexp.Compile("\n")
	if err != nil {
		fmt.Println("ERROR")
	}
	str = regex.ReplaceAllString(str, "")
	//fmt.Println("Get IP", str)
	return strings.TrimSpace(str)
}

func reboot() {
	cmdName := "reboot"
	cmdArgs := []string{"-n"}
	cmd := exec.Command(cmdName, cmdArgs...)
	err := cmd.Start()
	if err != nil {
		fmt.Println("reboot command wasn't able to be run")
	}
}

func main() {
	c := cron.New()
	c.AddFunc("0 * * * * *", func() {
		if getIP() == "" {
			// reboot()
			fmt.Println("NOTHING IN PARAMETERS")
		} else {
			fmt.Println("WE GOT AN ADDRESS AT PARAMETERS " + getIP())
		}
	})
	c.Start()

}
