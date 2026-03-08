//go:build linux

package hyproxia

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"

	"github.com/valyala/tcplisten"
)

func (p *Proxy) ListenPrefork(addr string) error {
	if os.Getenv(preforkWorkerEnv) != "" {
		// In worker process - set worker ID and PID for tracing
		p.workerPID = os.Getpid()
		workerIDStr := os.Getenv("HYPROXIA_WORKER_ID")
		p.workerID, _ = strconv.Atoi(workerIDStr)

		// Worker process - set GOMAXPROCS and serve
		if p.config.PreforkGOMAXPROCS > 0 {
			runtime.GOMAXPROCS(p.config.PreforkGOMAXPROCS)
		}

		cfg := tcplisten.Config{
			ReusePort:   true,
			DeferAccept: true,
			FastOpen:    true,
		}
		ln, err := cfg.NewListener("tcp4", addr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "worker listener error: %v\n", err)
			return err
		}
		return p.server.Serve(ln)
	}

	// Master process - determine number of workers
	numWorker := p.config.PreforkProcesses
	if numWorker <= 0 {
		numWorker = runtime.NumCPU()
	}

	maxProcs := p.config.PreforkGOMAXPROCS
	if maxProcs <= 0 {
		maxProcs = 2 // Most systems these days have hyperthreading, so 2 per core
	}

	// Startup message for prefork
	if !p.config.DisableStartupMessage {
		fmt.Printf("Prefork enabled: %d workers, %d GOMAXPROCS each\n", numWorker, maxProcs)
	}

	worker := make([]*exec.Cmd, numWorker)
	for i := range worker {
		cmd := exec.Command(os.Args[0], os.Args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = append(os.Environ(),
			preforkWorkerEnv+"=1",
			"HYPROXIA_WORKER_ID="+strconv.Itoa(i+1),
			"GOMAXPROCS="+strconv.Itoa(maxProcs),
		)
		if err := cmd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to start worker: %v\n", err)
			return err
		}
		worker[i] = cmd
	}

	// Print PIDs after all workers started
	if !p.config.DisableStartupMessage {
		fmt.Print("Worker PIDs: ")
		for i, cmd := range worker {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(cmd.Process.Pid)
		}
		fmt.Println()
	}

	// Wait for all workers
	for _, cmd := range worker {
		if err := cmd.Wait(); err != nil {
			fmt.Fprintf(os.Stderr, "worker exited with error: %v\n", err)
		}
	}
	return nil
}
