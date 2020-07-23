package main

import (
	goflag "flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/golang/glog"
	"github.com/opencontainers/selinux/go-selinux"
	"github.com/songgao/water"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	"golang.org/x/sys/unix"
)

var gid uint
var uid uint
var tapName string

func createTap(name string, isMultiqueue bool) error {
	tapDeviceArgs := []string{"tuntap", "add", "mode", "tap", "name", name}
	if isMultiqueue {
		tapDeviceArgs = append(tapDeviceArgs, "multi_queue")
	}
	cmd := exec.Command("ip", tapDeviceArgs...)
	err := cmd.Run()
	if err != nil {
		glog.Fatalf("Failed to create tap device %s. Reason: %v", name, err)
		return err
	}
	glog.Infof("Created tap device: %s", name)
	return nil
}

func createTapDevice(name string, uid uint, gid uint, isMultiqueue bool) error {
	var err error = nil
	config := water.Config{
		DeviceType: water.TAP,
		PlatformSpecificParams: water.PlatformSpecificParams{
			Name:    name,
			Persist: true,
			Permissions: &water.DevicePermissions{
				Owner: uid,
				Group: gid,
			},
			MultiQueue: isMultiqueue,
		},
	}

	_, err = water.New(config)
	return err
}

func createTapDeviceOnPIDNetNs(launcherPid string, tapName string, uid uint, gid uint) {
	netns, err := ns.GetNS(fmt.Sprintf("/proc/%s/ns/net", launcherPid))

	if err != nil {
		glog.Fatalf("Could not load netns: %+v", err)
	} else if netns != nil {
		glog.V(4).Info("Successfully loaded netns ...")

		err = netns.Do(func(_ ns.NetNS) error {
			desiredLabel := "system_u:system_r:container_t:s0"
			if err := selinux.SetExecLabel(desiredLabel); err != nil {
				glog.Errorf("Failed to set label: %s. Reason: %v", desiredLabel, err)
				return err
			}
			glog.V(4).Infof("Successfully set selinux label: %s", desiredLabel)

			label, err := selinux.ExecLabel()
			if err != nil {
				glog.Errorf("Failed to read label. Reason: %v", err)
			}
			glog.V(4).Infof("Read back the context: %s", label)
			if err := createTap(tapName, false); err != nil {
				glog.Fatalf("error creating tap device: %v", err)
			}

			glog.V(4).Infof("Managed to create the tap device in pid %s", launcherPid)
			return nil
		})
	}
}

func init() {
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
}

func main() {
	goflag.Parse()
	if err := flag.Set("alsologtostderr", "true"); err != nil {
		os.Exit(32)
	}

	rootCmd := &cobra.Command{
		Use: "tap-maker",
	}

	rootCmd.PersistentFlags().StringVar(&tapName, "tap-name", "tap0", "the name of the tap device")
	rootCmd.PersistentFlags().UintVar(&gid, "gid", 0, "the owner GID of the tap device")
	rootCmd.PersistentFlags().UintVar(&uid, "uid", 0, "the owner UID of the tap device")

	createTapCmd := &cobra.Command{
		Use:   "create-tap",
		Short: "create a tap device in a given PID net ns",
		RunE: func(cmd *cobra.Command, args []string) error {
			tapName := cmd.Flag("tap-name").Value.String()
			launcherPID := cmd.Flag("launcher-pid").Value.String()
			uidStr := cmd.Flag("uid").Value.String()
			gidStr := cmd.Flag("gid").Value.String()

			uid, err := strconv.ParseUint(uidStr, 10, 32)
			if err != nil {
				return err
			}
			gid, err := strconv.ParseUint(gidStr, 10, 32)
			if err != nil {
				return err
			}

			glog.V(4).Infof("Executing in netns of pid %s", launcherPID)
			createTapDeviceOnPIDNetNs(launcherPID, tapName, uint(uid), uint(gid))

			return nil
		},
	}

	createTapCmd.Flags().StringP("launcher-pid", "p", "", "specify the PID holding the netns where the tap device will be created")
	if err := createTapCmd.MarkFlagRequired("launcher-pid"); err != nil {
		os.Exit(1)
	}

	consumeTapCmd := &cobra.Command{
		Use:   "consume-tap",
		Short: "consume a tap device in the current net ns",
		RunE: func(cmd *cobra.Command, args []string) error {
			tapName := cmd.Flag("tap-name").Value.String()
			uidStr := cmd.Flag("uid").Value.String()
			gidStr := cmd.Flag("gid").Value.String()

			uid, err := strconv.ParseUint(uidStr, 10, 32)
			if err != nil {
				return err
			}
			gid, err := strconv.ParseUint(gidStr, 10, 32)
			if err != nil {
				return err
			}

			glog.V(4).Infof("Will consume tap device named: %s", tapName)
			err = createTapDevice(tapName, uint(uid), uint(gid), false)
			if err != nil {
				glog.Fatalf("Could not open the tapsy-thingy: %v", err)
			}

			glog.V(4).Infof("Opened the tap device on pid %d", os.Getpid())
			for {
				time.Sleep(time.Second)
			}
		},
	}

	execCmd := &cobra.Command{
		Use:   "exec",
		Short: "execute a sandboxed command in a specific mount namespace",
		Args:  cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			mntNamespace := cmd.Flag("mount").Value.String()
			if mntNamespace != "" {
				// join the mount namespace of a process
				fd, err := os.Open(mntNamespace)
				if err != nil {
					return fmt.Errorf("failed to open mount namespace: %v", err)
				}
				defer fd.Close()

				if err = unix.Unshare(unix.CLONE_NEWNS); err != nil {
					return fmt.Errorf("failed to detach from parent mount namespace: %v", err)
				}
				if err := unix.Setns(int(fd.Fd()), unix.CLONE_NEWNS); err != nil {
					return fmt.Errorf("failed to join the mount namespace: %v", err)
				}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			err := syscall.Exec(args[0], args, os.Environ())
			if err != nil {
				return fmt.Errorf("failed to execute command: %v", err)
			}

			return nil
		},
	}

	execCmd.Flags().StringP("mount", "m", "", "specify the mount namespace")
	if err := execCmd.MarkFlagRequired("mount"); err != nil {
		os.Exit(1)
	}

	rootCmd.AddCommand(createTapCmd, consumeTapCmd, execCmd)
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
