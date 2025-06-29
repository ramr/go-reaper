package main

import "encoding/json"
import "fmt"
import "os"
import "os/signal"
import "os/exec"
import "io/ioutil"
import "path/filepath"
import "syscall"
import "time"

import reaper "github.com/ramr/go-reaper"

const SCRIPT_THREADS_NUM = 10
const REAPER_JSON_CONFIG = "/reaper/config/reaper.json"
const NAME = "test-oop-pid1"

// Reaper test options.
type TestOptions struct {
	Pid              int
	Options          int
	DisablePid1Check bool
	Debug            bool
	TestStatus       bool
	TestStatusClose  bool
}

// Test with a bunch of different commands.
func commandTest() {
	dateCmd := exec.Command("date")

	dateOut, err := dateCmd.Output()
	if err != nil {
		panic(err)
	}
	fmt.Println("> date")
	fmt.Println(string(dateOut))

	grepCmd := exec.Command("grep", "hello")
	grepCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: 0}

	grepIn, _ := grepCmd.StdinPipe()
	grepOut, _ := grepCmd.StdoutPipe()
	grepCmd.Start()
	grepIn.Write([]byte("hello grep\ngoodbye grep"))
	grepIn.Close()
	grepBytes, _ := ioutil.ReadAll(grepOut)
	grepCmd.Wait()

	fmt.Println("> grep hello")
	fmt.Println(string(grepBytes))

	lsCmd := exec.Command("bash", "-c", "ls -a -l -h /tmp")
	lsOut, err := lsCmd.Output()
	if err != nil {
		os.Stderr.WriteString(err.Error())
	}
	fmt.Println("> ls -a -l -h /tmp")
	fmt.Println(string(lsOut))

	ls2Cmd := exec.Command("bash", "-c", "ls -a -l -h foo.adskdlskldk")
	ls2Cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: 0}
	out2, err2 := ls2Cmd.CombinedOutput()
	fmt.Println("> ls -a -l -h foo.adskdlskldk")
	if err2 != nil {
		os.Stderr.WriteString(err2.Error())
	}
	fmt.Println(string(out2))

	t1Cmd := exec.Command("bash", "-c", "/reaper/bin/oop-workers.sh")
	t1Cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: 0}
	out3, err3 := t1Cmd.CombinedOutput()
	fmt.Println("> /reaper/bin/oop-workers.sh")
	if err3 != nil {
		os.Stderr.WriteString(err3.Error())
	}
	fmt.Println(string(out3))

} /*  End of function  commandTest.  */

// Test with a process that sleeps for a short time.
func sleeperTest(set_proc_attributes bool) {
	fmt.Printf("%s: Set process attributes: %+v\n", NAME, set_proc_attributes)

	cmd := exec.Command("sleep", "1")
	if set_proc_attributes {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
			Pgid:    0,
		}
	}

	err := cmd.Start()
	if err != nil {
		fmt.Printf("%s: Error starting sleep command: %s\n", NAME, err)
		return
	}

	// Sleep for a wee bit longer to allow the reaper to reap the
	// command on a slow system.
	time.Sleep(4 * time.Second)

	err = cmd.Wait()
	if err != nil {
		if set_proc_attributes {
			fmt.Printf("%s: Error waiting for command: %s\n", NAME,
				err)
		} else {
			fmt.Printf("%s: Expected wait failure: %s\n", NAME, err)
		}
	}

} /*  End of function  sleeperTest.  */

// Start up test workers that in turn startup child processes, which will
// get orphaned.
func startWorkers() {
	//  Starts up workers - which in turn start up kids that get
	//  "orphaned".
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Printf("%s: Error getting script dir - %s\n", NAME, err)
		return
	}

	var scriptFile = fmt.Sprintf("%s/bin/script.sh", dir)
	script, err := filepath.Abs(scriptFile)
	if err != nil {
		fmt.Printf("%s: Error getting script - %s\n", NAME, scriptFile)
		return
	}

	var args = fmt.Sprintf("%d", SCRIPT_THREADS_NUM)
	var cmd = exec.Command(script, args)
	cmd.Start()

	fmt.Printf("%s: Started worker: %s %s\n", NAME, script, args)

	cmd.Wait()

} /*  End of function  startWorkers.  */

// Load reaper json and make test options.
func loadTestOptions() *TestOptions {
	configFile, err := os.Open(REAPER_JSON_CONFIG)
	if err != nil {
		fmt.Printf("%s: No reaper config: %v\n", NAME, err)
		return nil
	}

	options := TestOptions{Pid: -1, Options: 0, DisablePid1Check: false}
	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(&options)
	if err != nil {
		fmt.Printf("%s: Error in json config: %s\n", NAME, err)
		return nil
	}

	return &options

} /*  End of function  loadTestOptions.  */

// Start reaper.
func startReaper() {
	options := loadTestOptions()

	/*  Start the grim reaper ... */
	if options == nil {
		// No options, test reaper with the default config.
		fmt.Printf("%s: Using defaults ...\n", NAME)
		go reaper.Reap()
		return
	}

	config := reaper.Config{
		Pid:              options.Pid,
		Options:          options.Options,
		DisablePid1Check: options.DisablePid1Check,
		Debug:            options.Debug,
	}

	go reaper.Start(config)

	/*  Run the sleeper test setting the process attributes.  */
	go sleeperTest(true)

	/*  And run test without setting process attributes.  */
	go sleeperTest(false)

} /*  End of function startReaper.  */

// Launch the test processes.
func launchTest() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGUSR1)

	/*  Run the simple command test ... */
	commandTest()

	/*  Start the initial set of workers ... */
	startWorkers()

	for {
		select {
		case <-sig:
			fmt.Printf("%s: Got SIGUSR1, adding workers ...\n", NAME)
			startWorkers()
		}

	} /*  End of while doomsday ... */

} /*  End of function  launchTest.  */

// Print running processes.
func printProcesses() {
	cmd := exec.Command("bash", "-c", "ps -ef")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: 0}
	out, err := cmd.CombinedOutput()
	fmt.Println("> ps -ef ")
	if err != nil {
		os.Stderr.WriteString(err.Error())
	}
	fmt.Println(string(out))

} /*  End of function  printProcesses.  */

// Start reaper out-of-process.
func startReaperOutOfProcess() {
	fmt.Printf("in main pid = %d\n", os.Getpid())

	// Use an environment variable REAPER to indicate whether or not
	// we are the child/parent.
	if _, hasReaper := os.LookupEnv("REAPER"); !hasReaper {
		fmt.Println("in parent: starting reaper")
		startReaper()

		// Note: Optionally add an argument to the end to more
		//       easily distinguish the parent and child in
		//       something like `ps` etc.
		// args := os.Args
		args := append(os.Args, "#kiddo")

		pwd, err := os.Getwd()
		if err != nil {
			// Note: Better if you can handle it with a
			//       default directory ala "/tmp".
			panic(err)
		}

		kidEnv := []string{fmt.Sprintf("REAPER=%d", os.Getpid())}
		var wstatus syscall.WaitStatus
		pattrs := &syscall.ProcAttr{
			Dir: pwd,
			Env: append(os.Environ(), kidEnv...),
			Sys: &syscall.SysProcAttr{Setsid: true},
			Files: []uintptr{
				uintptr(syscall.Stdin),
				uintptr(syscall.Stdout),
				uintptr(syscall.Stderr),
			},
		}

		pid, _ := syscall.ForkExec(args[0], args, pattrs)

		fmt.Printf("kiddo-pid = %d\n", pid)

		_, err = syscall.Wait4(pid, &wstatus, 0, nil)
		for syscall.EINTR == err {
			_, err = syscall.Wait4(pid, &wstatus, 0, nil)
		}

		printProcesses()
		// If you put this code into a function, then exit here.
		os.Exit(0)
		return
	}

	fmt.Printf("in worker: my-pid = %d\n", os.Getpid())

} /*  End of function  startReaperOutOfProcess  */

// main out-of-process test entry point.
func main() {
	startReaperOutOfProcess()

	launchTest()

} /*  End of function  main.  */
