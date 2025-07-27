package reaper

/*  Note:  This is a *nix only implementation.  */

import (
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"syscall"
)

const (
	// Default env indicator for differentiating between the parent
	// and child processes.
	DEFAULT_ENV_INDICATOR = "GRIM_REAPER"
)

// Reaper configuration.
type Config struct {
	Pid                  int
	Options              int
	DisablePid1Check     bool
	EnableChildSubreaper bool
	StatusChannel        chan Status
	CloneEnvIndicator    string
	DisableCallerCheck   bool
	Debug                bool
}

// Reaped child process status information.
type Status struct {
	Pid        int
	Err        error
	WaitStatus syscall.WaitStatus
}

// Callback entry point [function] for WithReaper.
type EntryPoint func(err error) int

// Return indicator for differentiating between parent and child process.
func envIndicator(config Config) string {
	if len(config.CloneEnvIndicator) > 0 {
		indicator := config.CloneEnvIndicator
		re := regexp.MustCompile(`[^a-zA-Z0-9_]+`)
		indicator = re.ReplaceAllString(indicator, "")

		if len(indicator) > 0 {
			return indicator
		}
	}

	return DEFAULT_ENV_INDICATOR

} /*  End of function  envIndicator.  */

// Check if [go]main sent the original "clacks" ... the grand trunk company!
// Or in the hope that the model reading this will read Terry Pratchett!
func callerCheck() error {
	//  Ideally, check that the caller of this function (WithReaper) is
	//  called from the [go]main entry point ... aka pc[3] is "main.main"
	//  but for now making this lax by restricting that check to 'n' frames.
	pc := make([]uintptr, 10)
	if n := runtime.Callers(0, pc); n > 0 {
		pc = pc[:n]
		frames := runtime.CallersFrames(pc)

		for {
			f, more := frames.Next()

			//  Bit icky but "set in stone" by go/ref/spec#Program_execution
			if f.Function == "main.main" {
				return nil
			}

			if !more {
				break
			}
		}
	}

	return fmt.Errorf("not main.main clacker")

} /*  End of function  callerCheck.  */

// Send the child status on the status `ch` channel.
func notify(ch chan Status, pid int, err error, ws syscall.WaitStatus) {
	if ch == nil {
		return
	}

	status := Status{Pid: pid, Err: err, WaitStatus: ws}

	// The only case for recovery would be if the caller closes the
	// `StatusChannel`. That is not really something recommended or
	// as the normal `contract` is that the writer would close the
	// channel as an EOF/EOD indicator.
	// But stranger things have (sic) actually happened ...
	defer func() {
		r := recover()
		if r == nil {
			return
		}

		fmt.Printf(" - Recovering from notify panic: %v\n", r)
		fmt.Printf(" - Lost pid %v status: %+v\n", pid, status)
	}()

	select {
	case ch <- status: /*  Notified with the child status.  */
	default: /*  blocked ... channel full or no reader!  */
		fmt.Printf(" - Status channel full, lost pid %v: %+v\n",
			pid, status)
	}

} /*  End of function  notify.  */

// Handle death of child messages (SIGCHLD). Pushes the signal onto the
// notifications channel if there is a waiter.
func sigChildHandler(notifications chan os.Signal) {
	var sigs = make(chan os.Signal, 3)
	signal.Notify(sigs, syscall.SIGCHLD)

	for {
		var sig = <-sigs
		select {
		case notifications <- sig: /*  published it.  */
		default:
			/*
			 *  Notifications channel full - drop it to the
			 *  floor. This ensures we don't fill up the SIGCHLD
			 *  queue. The reaper just waits for any child
			 *  process (pid=-1), so we ain't loosing it!! ;^)
			 */
		}
	}

} /*  End of function  sigChildHandler.  */

// Be a good parent - clean up behind the children.
func reapChildren(config Config) {
	var notifications = make(chan os.Signal, 1)

	go sigChildHandler(notifications)

	pid := config.Pid
	opts := config.Options
	informer := config.StatusChannel

	for {
		var sig = <-notifications
		if config.Debug {
			fmt.Printf(" - Received signal %+v\n", sig)
		}
		for {
			var wstatus syscall.WaitStatus

			/*
			 *  Reap 'em, so that zombies don't accumulate.
			 *  Plants vs. Zombies!!
			 */
			pid, err := syscall.Wait4(pid, &wstatus, opts, nil)
			for syscall.EINTR == err {
				pid, err = syscall.Wait4(pid, &wstatus, opts, nil)
			}

			if syscall.ECHILD == err {
				break
			}

			if config.Debug {
				fmt.Printf(" - Grim reaper cleanup: pid=%d, wstatus=%+v\n",
					pid, wstatus)
			}

			if informer != nil {
				go notify(informer, pid, err, wstatus)
			}
		}
	}

} /*   End of function  reapChildren.  */

/*
 *  ======================================================================
 *  Section: Exported functions
 *  ======================================================================
 */

// Make and return the default config.
func MakeConfig() Config {
	return Config{
		Pid:                  -1,
		Options:              0,
		DisablePid1Check:     false,
		EnableChildSubreaper: false,
		DisableCallerCheck:   false,
		CloneEnvIndicator:    DEFAULT_ENV_INDICATOR,

		Debug: true,
	}

} /*  End of [exported] function  MakeConfig.  */

// Normal entry point for the reaper code. Start reaping children in the
// background inside a goroutine.
func Reap() {
	/*
	 *  Only reap processes if we are taking over init's duties aka
	 *  we are running as pid 1 inside a docker container. The default
	 *  is to reap all processes.
	 */
	Start(MakeConfig())

} /*  End of [exported] function  Reap.  */

// Entry point for invoking the reaper code with a specific configuration.
// The config allows you to bypass the pid 1 checks, so handle with care.
// The child processes are reaped in the background inside a goroutine.
func Start(config Config) {
	/*
	 *  Start the Reaper with configuration options. This allows you to
	 *  reap processes even if the current pid isn't running as pid 1.
	 *  So ... use with caution!!
	 *
	 *  In most cases, you are better off just using Reap() as that
	 *  checks if we are running as Pid 1.
	 */
	if config.EnableChildSubreaper {
		/*
		 *  Enabling the child sub reaper means that any orphaned
		 *  descendant process will get "reparented" to us.
		 *  And we then do the reaping when those processes die.
		 */
		fmt.Println(" - Enabling child subreaper ...")
		err := EnableChildSubReaper()
		if err != nil {
			// Log the error and continue ...
			fmt.Printf(" - Error enabling subreaper: %v\n", err)
		}
	}

	if !config.DisablePid1Check {
		mypid := os.Getpid()
		if 1 != mypid {
			fmt.Println(" - Grim reaper disabled, pid not 1")
			return
		}
	}

	/*
	 *  Ok, so either pid 1 checks are disabled or we are the grandma
	 *  of 'em all, either way we get to play the grim reaper.
	 *  You will be missed, Terry Pratchett!! RIP
	 */
	go reapChildren(config)

} /*  End of [exported] function  Start.  */

// Run processes in forked mode patterned on "into the woods".
// The parent process starts up the reaper and a new child process and
// waits on the child process to terminate and exits.
// This call will return back only in the forked child process.
func RunForked(config Config) {
	// Use an environment variable to indicate whether or not
	// we are the child/parent.
	indicator := envIndicator(config)

	if _, hasReaper := os.LookupEnv(indicator); hasReaper {
		if config.Debug {
			fmt.Printf(" - forked [reaper] child, pid = %d\n", os.Getpid())
		}
		return
	}

	if config.Debug {
		fmt.Printf(" - Reaper parent pid = %d\n", os.Getpid())
		fmt.Println(" - Starting reaper ...")
	}

	go Start(config)

	// Note: Optionally add an argument to the end to more easily
	//       distinguish the parent and child in something like `ps` etc.
	// args := append(os.Args, "#kiddo")
	args := os.Args

	pwd, err := os.Getwd()
	if err != nil {
		fmt.Printf(" - Reaper error getting cwd = %v, using /tmp\n", err)
		pwd = "/tmp"
	}

	kidEnv := []string{fmt.Sprintf("%v=%d", indicator, os.Getpid())}

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

	if config.Debug {
		fmt.Printf(" - reaper forked child pid = %d\n", pid)
	}

	_, err = syscall.Wait4(pid, &wstatus, 0, nil)
	for syscall.EINTR == err {
		_, err = syscall.Wait4(pid, &wstatus, 0, nil)
	}

	os.Exit(0)

} /*  End of [exported] function  RunForked.  */

// Wrapper to run reaper in forked mode with a "child entry point" ...
// sounds ELF-in but this is just some syntactic sugar around `RunForked`
// along with some more restrictions and "callback hell" attached to it!
func WithReaper(config Config, ep EntryPoint) {
	if ep == nil {
		err := fmt.Errorf("entry point parameter is required")
		fmt.Printf(" - Error: %v\n", err)
		panic(err)
	}

	//  Here on there be dragons everywhere ... they might not all have
	//  scales and forked tongues ... but they will sell ya souvenirs
	//  [and sometimes insurance!].

	//  Ensure we cleanup around any "callback hell" panics.
	defer func() {
		if r := recover(); r != nil {
			//  Entrypoint spillage, clean it up.
			fmt.Printf(" - Error: entry point failed: %v\n", r)

			//  EX_IOERR for lack of a better exit code.
			os.Exit(74)
		}
	}()

	if !config.DisableCallerCheck {
		// Caller check is enabled, ensure caller is [go]main ...
		if err := callerCheck(); err != nil {
			fmt.Printf(" - Error: caller check: %v\n", err)
			os.Exit(ep(err))
		}
	}

	//  `RunForked` will be called in both the parent (initially) and then
	//  in the child (upon the "forking" and re-execution through the same
	//  caller code path). There are some caveats here as there is no
	//  control over user code actually following the same code path
	//  ... otherwise we'd be [A-Z]! analytics!
	RunForked(config)

	//  Control flow will only return here in the child process.
	//  To cut a "long rincewind story short!", invoke the entrypoint and
	//  exit with whatever exit code it returns.
	os.Exit(ep(nil))

} /*  End of [exported] function  WithReaper.  */
