package main

import (
	"encoding/json"
	"fmt"
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()

	fmt.Printf("%s: Started worker: %s %s\n", NAME, script, args)

} /*  End of function  startWorkers.  */

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
				// Channel closed, no more work to do.
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
	}

	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(&options)
	if err != nil {
		fmt.Printf("%s: Error in json config: %s\n", NAME, err)
		return nil
	}

	return &options

} /*  End of function  loadTestOptions.  */

// Start reaper.
func startReaper(options *TestOptions) {
	/*  Start the grim reaper ... */
	if options == nil {
		// No options, test reaper with the default config.
		fmt.Printf("%s: Using defaults ...\n", NAME)
		go reaper.Reap()
		return
	}

	var statusChannel chan reaper.Status

	if options.Status {
		flag := options.StatusClose
		fmt.Printf("%s: status channel enabled, random close=%v\n",
			NAME, flag)

		// make a buffered channel with max 42 entries.
		statusChannel = make(chan reaper.Status, 42)
		go dumpChildExitStatus(statusChannel, flag)
	}

	config := reaper.Config{
		Pid:                  options.Pid,
		Options:              options.Options,
		DisablePid1Check:     options.DisablePid1Check,
		EnableChildSubreaper: options.EnableChildSubreaper,
		Debug:                options.Debug,
		StatusChannel:        statusChannel,
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

// main test entry point.
func main() {
	config := REAPER_JSON_CONFIG
	if len(os.Args) > 1 {
		config = os.Args[1]
	}

	options := loadTestOptions(config)
	startReaper(options)

	launchTest()

} /*  End of function  main.  */
