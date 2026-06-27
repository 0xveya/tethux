package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/moby/moby/client"
	"github.com/spf13/cobra"

	"github.com/0xveya/tethux/internal/libtethux/virt"
	"github.com/0xveya/tethux/internal/libtethux/virt/container"
	"github.com/0xveya/tethux/internal/libtethux/virt/container/podman"
)

func newPodman(socketOverride string) (*podman.Podman, error) {
	var opts []podman.Option
	if socketOverride != "" {
		opts = append(opts, podman.WithSocket(socketOverride))
	}
	return podman.New(opts...)
}

func smokeCmd() *cobra.Command {
	var (
		socket string
		name   string
		image  string
		cmd    []string
	)

	c := &cobra.Command{
		Use:   "smoke",
		Short: "spin up a container, exec into it, then clean up",
		RunE: func(c *cobra.Command, args []string) error {
			ctx := context.Background()

			p, err := newPodman(socket)
			if err != nil {
				return err
			}

			fmt.Printf("pulling %s...\n", image)
			if pullErr := p.Pull(ctx, image, &client.ImagePullOptions{}); pullErr != nil {
				return pullErr
			}
			fmt.Println("pulled", image)

			_ = p.Delete(ctx, name)

			node, err := p.CreateContainer(ctx, &container.ContainerConfig{
				NodeConfig: virt.NodeConfig{
					Name:  name,
					Image: image + ":latest",
				},
				Cmd: cmd,
			})
			if err != nil {
				return err
			}
			fmt.Println("created", node.ID[:12], "name:", node.Name)

			defer func() {
				fmt.Println("cleaning up...")
				if deleteErr := p.DeleteContainer(ctx, node.ID, &client.ContainerRemoveOptions{Force: true}); deleteErr != nil {
					fmt.Println("cleanup error:", deleteErr)
				} else {
					fmt.Println("deleted", node.ID[:12])
				}
			}()

			if startErr := p.Start(ctx, node.ID); startErr != nil {
				return startErr
			}

			state, statErr := p.State(ctx, node.ID)
			if statErr != nil {
				return statErr
			}
			fmt.Println("state:", state)

			node, err = p.Inspect(ctx, node.ID, nil)
			if err != nil {
				return err
			}
			fmt.Printf("node: id=%s name=%s state=%s image=%s networks=%v\n",
				node.ID[:12], node.Name, node.State, node.ImageName, node.Networks)

			stdout, stderr, err := p.Exec(ctx, node.ID, []string{"echo", "meow"}, nil, nil)
			if err != nil {
				return err
			}
			fmt.Println("exec stdout:", string(stdout))
			if len(stderr) > 0 {
				fmt.Println("exec stderr:", string(stderr))
			}

			reader, err := p.Logs(ctx, node.ID, nil)
			if err != nil {
				return err
			}
			fmt.Println("logs:")
			_, _ = io.Copy(os.Stdout, reader)
			_ = reader.Close()
			fmt.Println()

			if err := p.SuspendContainer(ctx, node.ID, nil); err != nil {
				return fmt.Errorf("suspend: %w", err)
			}
			fmt.Println("suspended")

			if err := p.ResumeContainer(ctx, node.ID, nil); err != nil {
				return fmt.Errorf("resume: %w", err)
			}
			fmt.Println("resumed")

			if err := p.RestartContainer(ctx, node.ID, nil); err != nil {
				return fmt.Errorf("restart: %w", err)
			}
			fmt.Println("restarted")

			if err := p.StopContainer(ctx, node.ID, nil); err != nil {
				return fmt.Errorf("stop: %w", err)
			}
			fmt.Println("stopped")

			return nil
		},
	}

	c.Flags().StringVar(&socket, "socket", "", "override podman socket path")
	c.Flags().StringVar(&name, "name", "tethux-smoke", "container name")
	c.Flags().StringVar(&image, "image", "alpine", "image to pull and run")
	c.Flags().StringSliceVar(&cmd, "cmd", []string{"sh", "-c", "echo meow && sleep 30"}, "command to run in container")

	return c
}

func listCmd() *cobra.Command {
	var socket string

	c := &cobra.Command{
		Use:   "list",
		Short: "list all containers",
		RunE: func(c *cobra.Command, args []string) error {
			ctx := context.Background()

			p, err := newPodman(socket)
			if err != nil {
				return err
			}

			nodes, err := p.List(ctx)
			if err != nil {
				return err
			}

			if len(nodes) == 0 {
				fmt.Println("no containers")
				return nil
			}

			fmt.Printf("%-14s %-20s %s\n", "ID", "NAME", "STATE")
			for _, n := range nodes {
				fmt.Printf("%-14s %-20s %s\n", n.ID[:12], n.Name, n.State)
			}
			return nil
		},
	}

	c.Flags().StringVar(&socket, "socket", "", "override podman socket path")
	return c
}

func pullCmd() *cobra.Command {
	var socket string

	c := &cobra.Command{
		Use:   "pull <ref>",
		Short: "pull an image",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			ctx := context.Background()

			p, err := newPodman(socket)
			if err != nil {
				return err
			}

			fmt.Printf("pulling %s...\n", args[0])
			if err := p.Pull(ctx, args[0], nil); err != nil {
				return err
			}
			fmt.Println("done")
			return nil
		},
	}

	c.Flags().StringVar(&socket, "socket", "", "override podman socket path")
	return c
}

func logsCmd() *cobra.Command {
	var (
		socket     string
		follow     bool
		timestamps bool
		tail       string
	)

	c := &cobra.Command{
		Use:   "logs <container-id>",
		Short: "fetch logs from a container",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			ctx := context.Background()

			p, err := newPodman(socket)
			if err != nil {
				return err
			}

			reader, err := p.Logs(ctx, args[0], &client.ContainerLogsOptions{
				Follow:     follow,
				Timestamps: timestamps,
				Tail:       tail,
			})
			if err != nil {
				return err
			}
			defer reader.Close()

			_, _ = io.Copy(os.Stdout, reader)
			return nil
		},
	}

	c.Flags().StringVar(&socket, "socket", "", "override podman socket path")
	c.Flags().BoolVarP(&follow, "follow", "f", false, "follow log output")
	c.Flags().BoolVarP(&timestamps, "timestamps", "t", false, "show timestamps")
	c.Flags().StringVar(&tail, "tail", "all", "number of lines from end")

	return c
}
