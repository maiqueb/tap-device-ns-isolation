package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/golang/glog"
	"github.com/songgao/water"
)

func createTapDevice(name string, isMultiqueue bool) error {
	var err error = nil
	config := water.Config{
		DeviceType: water.TAP,
		PlatformSpecificParams: water.PlatformSpecificParams{
			Name:    name,
			Persist: true,
			Permissions: &water.DevicePermissions{
				Owner: 107,
				Group: 107,
			},
			MultiQueue: isMultiqueue,
		},
	}

	_, err = water.New(config)
	return err
}

func setupLogger() {
	if err := flag.Set("logtostderr", "true"); err != nil {
		os.Exit(1)
	}
}

func main() {
	setupLogger()

	tapName := flag.String("tap-name", "tap0", "override the name of the tap device")
	launcherPid := flag.String("launcher-pid", "", "optionally specify the PID holding the netns where the tap device will be created.")
	flag.Parse()
	glog.V(4).Info("Started app")

	if *launcherPid != "" {
		glog.V(4).Infof("Executing in netns of pid %s", *launcherPid)
		netns, err := ns.GetNS(fmt.Sprintf("/proc/%s/ns/net", *launcherPid))

		if err != nil {
			glog.Fatalf("Could not load netns: %+v", err)
		} else if netns != nil {
			glog.V(4).Info("Successfully loaded netns ...")

			err = netns.Do(func(_ ns.NetNS) error {
				if err := createTapDevice(*tapName, false); err != nil {
					glog.Fatalf("error creating tap device: %v", err)
				}

				glog.V(4).Infof("Managed to create the tap device in pid %s", *launcherPid)
				return nil
			})
		}
	} else {
		if err := createTapDevice(*tapName, false); err != nil {
			glog.Fatalf("error creating tap device: %v", err)
		}
	}
	glog.V(4).Info("All good in tha hood")
	glog.Flush()
}
