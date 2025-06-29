package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	reaper "github.com/ramr/go-reaper"
)

const SCRIPT_THREADS_NUM = 3
const REAPER_JSON_CONFIG = "/reaper/config/reaper.json"
const NAME = "testpid1"

// Reaper test options.
type TestOptions struct {
	Pid                  int
	Options              int
	DisablePid1Check     bool
	EnableChildSubreaper bool
	Debug                bool
	Status               bool
	StatusClose          bool
	ErrorExit            bool
	RunForked            bool
	WithReaper           bool
	WithReaperOption     string
}

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

	//  Sleep for a wee bit longer to allow the reaper to reap the
	//  command on a slow system.
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

// Run command with bash -c ...
func runTestCommand(cmd string, set_proc_attrs bool) {
	command := exec.Command("bash", "-c", cmd)
	if set_proc_attrs {
		command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: 0}
	}

	fmt.Printf("> bash -c '%v'\n", cmd)

	zout, zerr := command.CombinedOutput()
	if zerr != nil {
		os.Stderr.WriteString(zerr.Error())
	}

	fmt.Println(string(zout))

} /*  End of function  runTestCommand.  */

// Test with a bunch of different commands.
func runCommandMix() {
	commands := []string{"ls -a -l -h /tmp", "ls -a -l -h foo.adskdlskldk",
		"date", "uname -srv", "whoami", "id", "stat /etc/hostname",
		"cat /etc/hostname", "cat /etc/passwd | wc -c",
		"cat /etc/passwd | cut -f 1 -d ':' |   sort   | head -n 2",
	}

	for _, cmd := range commands {
		runTestCommand(cmd, false)
		runTestCommand(cmd, true)
	}

	grepCmd := exec.Command("grep", "hello")
	grepCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: 0}

	grepIn, _ := grepCmd.StdinPipe()
	grepOut, _ := grepCmd.StdoutPipe()
	grepCmd.Start()
	grepIn.Write([]byte("hello grepped\ngoodbye\naloha\nciao\nhi\n\n"))
	grepIn.Close()
	grepBytes, _ := ioutil.ReadAll(grepOut)
	grepCmd.Wait()

	fmt.Println("> grep hello")
	fmt.Println(string(grepBytes))

} /*  End of function  runCommandMix.  */

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
		fmt.Printf("%s: Error getting script - %v\n", NAME, scriptFile)
		return
	}

	var args = fmt.Sprintf("%d", SCRIPT_THREADS_NUM)
	var cmd = exec.Command(script, args)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: 0}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()

	fmt.Printf("%s: Started workers via: %v %v\n", NAME, script, args)

	cmd.Wait()

	fmt.Printf("%v: Script %v completed\n", NAME, script)

} /*  End of function  startWorkers.  */

// Start the test launcher and start an initial set of workers.
func startLauncher() {
	fmt.Printf("%v: starting launcher ...\n", NAME)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGUSR1)

	/*  Run a few test commands.  */
	runCommandMix()

	/*  Start the initial set of detached workers ... */
	fmt.Printf("%v: starting detached workers ...\n", NAME)
	startWorkers()

	for {
		select {
		case <-sig:
			fmt.Printf("%s: Got SIGUSR1, adding detached workers\n", NAME)
			startWorkers()
		}

	} /*  End of while doomsday ... */

} /*  End of function  startLauncher.  */

// Start the test processes.
func startTestProcesses() {
	time.Sleep(1 * time.Second)

	/*  Run the sleeper test setting the process attributes.  */
	go sleeperTest(true)

	/*  And run test without setting process attributes.  */
	go sleeperTest(false)

	startLauncher()

} /*  End of function  startTestProcesses.  */

// Dump exit status of child processes. If flag is set it randomly tests
// closing the status channel.
func dumpChildExitStatus(channel chan reaper.Status, flag bool) {
	if channel == nil {
		return
	}

	nreaped := 0
	maxNotifications := 42 + rand.Intn(42)

	for {
		select {
		case status, ok := <-channel:
			if !ok {
				/*  Channel closed, no more work to do.   */
				fmt.Printf("%v: status channel closed\n", NAME)
				return
			}

			nreaped += 1
			exitCode := status.WaitStatus.ExitStatus()

			fmt.Printf("%v: status of pid %v, exit code %v\n",
				NAME, status.Pid, exitCode)

			if flag && nreaped > maxNotifications {
				close(channel)
				fmt.Printf("%v: random channel close\n",
					NAME)
			}
		}

	} /*  End of while doomsday ...  */

} /*  End of function  dumpChildExitStatus.  */

// Load reaper json and make test options.
func loadTestOptions(config string) *TestOptions {
	configFile, err := os.Open(config)
	if err != nil {
		fmt.Printf("%s: No reaper config: %v\n", NAME, err)
		return nil
	}

	options := TestOptions{
		Pid:              -1,
		Options:          0,
		DisablePid1Check: false,
		WithReaperOption: "",
	}

	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(&options)
	if err != nil {
		fmt.Printf("%s: Error in json config: %s\n", NAME, err)
		return nil
	}

	return &options

} /*  End of function  loadTestOptions.  */

// Return the test scenario based on the options.
func getTestScenario(options *TestOptions) string {
	if options == nil {
		return "default"
	}

	if options.RunForked {
		return "forked"
	}

	if options.WithReaper {
		return "swaddled" /*  aka a wrapped child.  */
	}

	return "options"

} /*  End of function  getTestScenario.  */

func configure(options *TestOptions) reaper.Config {
	var statusChannel chan reaper.Status

	if options.Status {
		flag := options.StatusClose
		fmt.Printf("%s: status channel enabled, random close=%v\n",
			NAME, flag)

		/*  Make a buffered channel with max 42 entries.  */
		statusChannel = make(chan reaper.Status, 42)
		go dumpChildExitStatus(statusChannel, flag)
	}

	return reaper.Config{
		Pid:                  options.Pid,
		Options:              options.Options,
		DisablePid1Check:     options.DisablePid1Check,
		EnableChildSubreaper: options.EnableChildSubreaper,
		Debug:                options.Debug,
		StatusChannel:        statusChannel,
		CloneEnvIndicator:    "_REAPER_TEST",
		DisableCallerCheck:   false,
	}

} /*  End of function  configure.  */

// Test default Reap method.
func testReap() {
	fmt.Printf("%s: Using defaults ...\n", NAME)

	/*  No options, test reaper with the default config.  */
	go reaper.Reap()

	startLauncher()

} /*  End of function  testReap.  */

// Wait for termination signal (SIGTERM).
func waitForTerminationSignal() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM)

	for {
		select {
		case <-sig:
			fmt.Printf("%s: Got termination signal, exiting ...\n", NAME)
			return
		}

	} /*  End of while doomsday ... */

} /*  End of function  waitForTerminationSignal.  */

// Test reaper started in forked mode.
func testRunForked(options *TestOptions) {
	config := configure(options)

	expectedCode := 0
	if options.ErrorExit {
		/*  This test needs the child to exit on receiving SIGTERM.  */
		/*  So generate a random exit code between 64-78.  */
		expectedCode = 64 + rand.Intn(15)
	}

	reaper.RunForked(config)

	/*  This will run only in the child process.  */
	go startTestProcesses()

	waitForTerminationSignal()
	fmt.Printf("%s: Exiting with code = %v\n", NAME, expectedCode)
	os.Exit(expectedCode)

} /*  End of function  testRunForked.  */

// Run reaper started in "swaddled" mode.
func runWithReaper(options *TestOptions) {
	expectedCode := 0
	if options.ErrorExit {
		/*  This test needs the child to exit on receiving SIGTERM.  */
		/*  So generate a random exit code between 64-78.  */
		expectedCode = 64 + rand.Intn(15)
	}

	config := configure(options)

	reaper.WithReaper(config, func(err error) int {
		if err != nil {
			fmt.Printf("%s: WithReaper error = %v\n", NAME, err)

			if options.WithReaperOption == "not-via-main" {
				fmt.Printf("%s: WithReaper not-via-main check OK\n", NAME)
				return 0
			}

			// Huh? Return EX_SOFTWARE, some internal error.
			fmt.Printf("%s: WithReaper unexpected error\n", NAME)
			fmt.Printf("%s: WithReaper exit code 70\n", NAME)
			return 70
		}

		go startTestProcesses()

		waitForTerminationSignal()
		if options.WithReaperOption == "child-panic" {
			fmt.Printf("%s: WithReaper child-panic test!\n", NAME)
			panic("test WithReaper child panic option")
		}

		fmt.Printf("%s: WithReaper exit code %v\n", NAME, expectedCode)
		return expectedCode
	})

	//  Should never reach here as WithReaper calls exit.
	msg := "POST WithReaper - should _NEVER_ reach here!"
	fmt.Printf("%s: %s\n", NAME, msg)
	panic(msg)

} /*  End of function  runWithReaper.  */

// Test WithReaper process.
func testWithReaper(options *TestOptions) {
	if options.WithReaperOption == "not-via-main" {
		go runWithReaper(options)

		waitForTerminationSignal()

		os.Exit(0) // or we can just return.
		return
	}

	runWithReaper(options)

} /*  End of function  testWithReaper.  */

// Test reaper start with config.
func testStartReaper(options *TestOptions) {
	/*  Start the grim reaper ... */
	config := configure(options)

	go reaper.Start(config)

	startTestProcesses()

} /*  End of function testStartReaper.  */

// main test entry point.
func main() {
	config := REAPER_JSON_CONFIG
	if len(os.Args) > 1 {
		config = os.Args[1]
	}

	options := loadTestOptions(config)

	switch getTestScenario(options) {
	case "default":
		testReap()

	case "forked":
		testRunForked(options)

	case "swaddled":
		testWithReaper(options)

	case "options":
		testStartReaper(options)

	default:
		testStartReaper(options)
	}

} /*  End of function  main.  */
