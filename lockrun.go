package lockrun

import (
	"fmt"
	"github.com/ogier/pflag"
	"log"
	"os"
	"os/exec"
	"time"
	"syscall"
)

var (
	LockFile  = pflag.StringP("lockfile", "L", "", "Specify the name of a file which is used for locking. This filename is created if necessary (with mode 0666), and no I/O of any kind is done. This file is never removed.")
	WaitLock  = pflag.BoolP("wait", "W", false, "When a lock is hit, we normally exit with error, but --wait causes it to loop until the lock is released.")
	SleepTime = pflag.IntP("sleep", "S", 10, "When using --wait, will sleep <sleep> seconds between attempts to acquire the lock.")
	Retries   = pflag.IntP("retries", "R", 0, "Attempt <retries> retries in each wait loop. 0 mean infinite loop.")
	Quiet     = pflag.BoolP("quiet", "Q", false, "Exit quietly (and with success) if locked.")
	Verbose   = pflag.BoolP("verbose", "V", false, "Show a bit more runtime debugging")
	MaxTime = pflag.Int("maxtime",0, "Wait for at most <maxtime> seconds for a lock, then exit. 0 mean wait infinite.")
)

func Main() {
	pflag.Usage = usage
	startTime := time.Now()
	pflag.Parse()

	if len(os.Args) < 2 {
		pflag.Usage()
		die("")
	}

	if *LockFile == "" {
		log.Fatalln("ERROR: missing --lockfile=F parameter")
	}

	var subProcessArgs []string
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i-1] != "--" {
			continue
		}
		subProcessArgs = os.Args[i:]
	}
	if len(subProcessArgs) == 0 {
		die("ERROR: missing command to %s (must follow \"--\" marker) ", os.Args[0])
	}

	var attempts int = 0

	if err := CheckCanLock(*LockFile); err != nil {
		die("ERROR: cannot open(%s) [err=%s]", *LockFile, err)
	}

	for !WaitAndLock(*LockFile) {
		attempts++
		if !*WaitLock {
			if *Quiet {
				return
			} else {
				die("ERROR: cannot launch %s - run is locked", subProcessArgs[0])
			}
		}
		if *Retries > 0 && attempts > *Retries {
			die("ERROR: cannot launch %v - run is locked (after %v attempts)", os.Args[0], attempts)
		}

		// Waiting
		if *Verbose {
			fmt.Fprintf(os.Stderr, "(locked: sleeping %d secs, after attempt #%d)\n", *SleepTime, attempts)
		}
		time.Sleep(time.Duration(*SleepTime) * time.Second)
	}

	cmd := exec.Command(subProcessArgs[0], subProcessArgs[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		die("ERROR: cannot fork [%v]", err);
	}
	if *Verbose {
		fmt.Fprintf(os.Stderr, "Waiting for process %v\n", cmd.Process.Pid)
	}
	cmd.Wait()
	exitCode := cmd.ProcessState.Sys().(syscall.WaitStatus).ExitCode

	executeDuration := time.Now().Sub(startTime)
	if *Verbose && *MaxTime > 0 && executeDuration > time.Duration(*MaxTime) * time.Second{
		fmt.Fprintf(os.Stderr, "pid %d exited with status %d (time=%v sec)\n", cmd.Process.Pid, exitCode, executeDuration)
	}
	if err != nil {
		die("ERROR: cannot fork [%s]", err)
	}
	if *Verbose {
		fmt.Printf("Waiting for process %v\n", cmd.Process.Pid)
	}
	Unlock()
	os.Exit(int(exitCode))
}

func usage (){
	fmt.Printf("%v Usage: lockrun [options] -- command args...\n", os.Args[0])
	pflag.PrintDefaults()
}

func die(format string, args ...interface{}) {
	if len(args) == 0 {
		os.Stderr.WriteString(format)
	} else {
		fmt.Fprintf(os.Stderr, format, args...)
	}
	os.Stderr.WriteString("\n")
	Unlock()
	os.Exit(1)
}
