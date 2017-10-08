package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
	// "github.com/robfig/cron"
)

var InitialTries int
var HasIP bool

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

func cronfunc() {
	fmt.Println(getIP())
	currentIP := getIP()
	// Since its in rc.local, we want the system networking to set everything up first (so give it some leway to make sure it doesn't fail just because it ran before wlan0 was ifuped)

	if HasIP { // Means at one point, an IP was assigned. If an IP was assigned, and there is no IP currently, then assume DHCP screwed up and the system needs a reboot
		if currentIP == "" {
			reboot()
		}
	} else {
		if InitialTries < 5 { // Try to wait for the networking system daemon to bring up all the interfaces before assuming failure
			if currentIP == "" { // If its reasonable to assume the network is still ifuping wlan0 and the IP is still blank, then add 1 to the InitialTries counter
				InitialTries += 1
			} else { // However, it its not "" (blank), then we can assume the DHCP server assigned us a usable IP, hense, the regular operation of the program can start.
				InitialTries = 5
				HasIP = true
			}
			// After 5 tries, its safe to assume bringing up the network interface for wlan0 failed or succeeded, and waiting more will not help the situation
		} else {
			if currentIP == "" { // If after 5 tries, the system can't get an IP address assigned, it means something is wrong with configuration, Hence, to stop an infinite reboot loop, it will just exit the program gracefully
				os.Exit(1)
			} else { // If on the 5th try, we get an actual non-empty IP, then HasIP becomes true and program starts regular behavior
				HasIP = true
			}
		}
	}

}

func main() {
	InitialTries = 0
	if getIP() == "" {
		HasIP = false
	} else {
		HasIP = true
	}
	// c := cron.New()
	// c.AddFunc("0 * * * * *", func() { cronfunc() })
	// c.Start()

	go func() {

		c := time.Tick(1 * time.Second)

		for range c {
			cronfunc()
		}

	}()

	time.Sleep(30 * time.Second)
}
