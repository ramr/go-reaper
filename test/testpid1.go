package main

import (
	"encoding/json"
	"fmt"
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

func sleeper_test(set_proc_attributes bool) {
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

} /*  End of function  sleeper_test.  */

func start_workers() {
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

} /*  End of function  start_workers.  */

func start_reaper() {
	useConfig := false
	var onReap reaper.ReapCallback = func(pid int, wstatus syscall.WaitStatus) {
		fmt.Printf("%s: Child process %d exited with code %d\n", NAME, pid, wstatus.ExitStatus())
	}

	defaultConfig := reaper.Config{
		Pid:              -1,
		Options:          0,
		DisablePid1Check: false,
		OnReap:           onReap,
	}
	config := defaultConfig

	configFile, err := os.Open(REAPER_JSON_CONFIG)
	if err == nil {
		decoder := json.NewDecoder(configFile)
		err = decoder.Decode(&config)
		if err == nil {
			fmt.Printf("%s: Using config %s\n", NAME,
				REAPER_JSON_CONFIG)
			fmt.Printf("%s: Make chan\n", NAME)
			config.OnReap = onReap
			useConfig = true
		} else {
			fmt.Printf("%s: Error in json config: %s\n", NAME, err)
			fmt.Printf("%s: Using defaults ...\n", NAME)
		}
	}

	/*  Start the grim reaper ... */
	if useConfig {
		go reaper.Start(config)

		/*  Run the sleeper test setting the process attributes.  */
		go sleeper_test(true)

		/*  And run test without setting process attributes.  */
		go sleeper_test(false)

	} else {
		go reaper.Start(defaultConfig)
		// go reaper.Reap()
	}

} /*  End of function start_reaper.  */

func launch_test() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGUSR1)

	/*  Start the initial set of workers ... */
	start_workers()

	for {
		select {
		case <-sig:
			fmt.Printf("%s: Got SIGUSR1, adding workers ...\n", NAME)
			start_workers()
		}

	} /*  End of while doomsday ... */

} /*  End of function  launch_test.  */

func main() {
	start_reaper()
	launch_test()

} /*  End of function  main.  */
