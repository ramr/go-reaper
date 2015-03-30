# go-reaper
Process (grim) reaper library for golang - this is useful for cleaning up
zombie processes inside docker containers (which do not have an init
process running as pid 1).


tl;dr
-----

       import reaper "github.com/ramr/go-reaper"

       func main() {
		//  Start background reaping of orphaned child processes.
		go reaper.Reap()

		//  Rest of your code ...
       }



How and Why
-----------
If you run a container without an init process (pid 1) which reaps zombie
processes and you have a program that does not clean up behind the kids,
you will end up with a lot of zombie processes and eventually exhaust the
max processes limit.  
  
This library allows a go program to setup a background signal handling
mechanism to handle the death of those orphaned children and not create
a load of zombies inside the pid namespace your container runs in.


Usage:
------
See the tl;dr section above.

