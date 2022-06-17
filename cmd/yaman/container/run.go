package container

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/yaman"
	"github.com/willdurand/containers/internal/yaman/container"
	"github.com/willdurand/containers/internal/yaman/shim"
	"golang.org/x/term"
)

func init() {
	cmd := &cobra.Command{
		Use:   "run <image> [<command> [<args>...]]",
		Short: "Run a command in a new container",
		Run:   cli.HandleErrors(run),
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.Flags().BoolP("detach", "d", false, "run container in background and print container ID")
	cmd.Flags().String("hostname", "", "set the container hostname")
	cmd.Flags().BoolP("interactive", "i", false, "keep stdin open")
	cmd.Flags().String("name", "", "assign a name to the container")
	cmd.Flags().Bool("rm", false, "automatically remove the container when it exits")
	cmd.Flags().String("runtime", "", "runtime to use for this container")
	cmd.Flags().BoolP("tty", "t", false, "allocate a pseudo-tty")
	containerCommand.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	rootDir, _ := cmd.Flags().GetString("root")
	interactive, _ := cmd.Flags().GetBool("interactive")
	detach, _ := cmd.Flags().GetBool("detach")

	// container options
	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		rand.Seed(time.Now().UnixNano())
		name = namesgenerator.GetRandomName(0)
	}
	rm, _ := cmd.Flags().GetBool("rm")
	hostname, _ := cmd.Flags().GetString("hostname")
	tty, _ := cmd.Flags().GetBool("tty")
	containerOpts := container.ContainerOpts{
		Name:     name,
		Command:  args[1:],
		Remove:   rm,
		Hostname: hostname,
		Tty:      tty,
	}

	// shim options
	shimOpts := shim.ShimOpts{}
	if runtime, _ := cmd.Flags().GetString("runtime"); runtime != "" {
		shimOpts.Runtime = runtime
	}

	containerId, err := yaman.Run(rootDir, args[0], containerOpts, shimOpts)
	if err != nil {
		return err
	}

	// In detached mode, we print the container ID to the standard output and we
	// are done. The container should be running as long as it is supposed to
	// (e.g., if the command exits after completion, the container might be
	// exited but if the command is a daemon, the container should still be
	// alive).
	if detach {
		fmt.Fprintln(os.Stdout, containerId)
		return nil
	}

	// When we are not in a detached mode, there is more work... so we need to
	// get the shim instance because there is some IO-related stuff to set up.
	shim, err := shim.Load(rootDir, containerId)
	if err != nil {
		return err
	}

	stdin, stdout, stderr, err := shim.OpenStreams()
	if err != nil {
		return err
	}
	defer stdin.Close()
	defer stdout.Close()
	defer stderr.Close()

	// In interactive mode, we keep `stdin` open, otherwise we close it
	// immediately and only care about `stdout` and `stderr`.
	if interactive {
		go io.Copy(stdin, os.Stdin)
	} else {
		stdin.Close()
	}

	if containerOpts.Tty {
		// We force the current terminal to switch to "raw mode" because we don't
		// want it to mess with the PTY set up by the container itself.
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			return err
		}
		defer term.Restore(int(os.Stdin.Fd()), oldState)

		go io.Copy(stdin, os.Stdin)
		// Block on the stream coming from the container so that when it exits, we
		// can also exit this command.
		io.Copy(os.Stdout, stdout)
	} else {
		var wg sync.WaitGroup
		// We copy the data from the container to the appropriate streams as long
		// as we can. When the container process exits, the shimm should close the
		// streams on its end, which should allow `copyStd()` to complete.
		wg.Add(1)
		go copyStd(stdout, os.Stdout, &wg)

		wg.Add(1)
		go copyStd(stderr, os.Stderr, &wg)

		wg.Wait()
	}

	return nil
}

func copyStd(s *os.File, w io.Writer, wg *sync.WaitGroup) {
	defer wg.Done()

	reader := bufio.NewReader(s)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		fmt.Fprint(w, line)
	}
}
