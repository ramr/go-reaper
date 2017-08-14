package main

import "encoding/json"
import "fmt"
import "os"
import "os/signal"
import "os/exec"
import "path/filepath"
import "syscall"
import "time"

import reaper "github.com/ramr/go-reaper"

const NWORKERS = 3
const REAPER_JSON_CONFIG = "/reaper/config/reaper.json"

func sleeper_test(set_proc_attributes bool) {
	fmt.Printf(" - Set process attributes: %+v\n", set_proc_attributes)

	cmd := exec.Command("sleep", "1")
	if set_proc_attributes {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
			Pgid:    0,
		}
	}

	err := cmd.Start()
	if err != nil {
		fmt.Printf(" - Error starting sleep command: %s\n", err)
		return
	}

	fmt.Printf("Set proc attributes: %+v\n", set_proc_attributes)

	// Sleep for a wee bit longer to allow the reaper to reap the
	// command on a slow system.
	time.Sleep(4 * time.Second)

	err = cmd.Wait()
	if err != nil {
		if set_proc_attributes {
			fmt.Printf(" - Error waiting for command: %s\n",
				err)
		} else {
			fmt.Printf(" - Expected wait failure: %s\n", err)
		}
	}

} /*  End of function  sleeper_test.  */

func start_workers() {
	//  Starts up workers - which in turn start up kids that get
	//  "orphaned".
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Printf(" - Error getting script dir - %s\n", err)
		return
	}

	var scriptFile = fmt.Sprintf("%s/bin/script.sh", dir)
	script, err := filepath.Abs(scriptFile)
	if err != nil {
		fmt.Printf(" - Error getting script - %s\n", scriptFile)
		return
	}

	var args = fmt.Sprintf("%d", NWORKERS)
	var cmd = exec.Command(script, args)
	cmd.Start()

	fmt.Printf("  - Started worker: %s %s\n", script, args)

} /*  End of function  start_workers.  */

func main() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGUSR1)

	useConfig := false
	config := reaper.Config{}

	configFile, err := os.Open(REAPER_JSON_CONFIG)
	if err == nil {
		decoder := json.NewDecoder(configFile)
		err = decoder.Decode(&config)
		if err == nil {
			useConfig = true
		} else {
			fmt.Printf(" - Error: Invalid json config: %s", err)
			fmt.Printf(" - Using defaults ... ")
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
		go reaper.Reap()
	}

	/*  Start the initial set of workers ... */
	start_workers()

	for {
		select {
		case <-sig:
			fmt.Println("  - Got SIGUSR1, adding workers ...")
			start_workers()
		}

	} /*  End of while doomsday ... */

} /*  End of function  main.  */
