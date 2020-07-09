package main

import (
	"flag"
	"fmt"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/golang/glog"
	"github.com/songgao/water"
	"os"
)

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

func setupLogger() {
	if err := flag.Set("logtostderr", "true"); err != nil {
		os.Exit(1)
	}
}

func main() {
	setupLogger()

	tapName := flag.String("tap-name", "tap0", "override the name of the tap device")
	launcherPid := flag.String("launcher-pid", "", "optionally specify the PID holding the netns where the tap device will be created.")
	uid := flag.Int("uid", 0, "the owner UID of the tap device")
	gid := flag.Int("gid", 0, "the owner GID of the tap device")

	flag.Parse()
	appMode := "create-tap"
	glog.V(4).Infof("Started app in %s mode", appMode)

	if *launcherPid != "" {
		glog.V(4).Infof("Executing in netns of pid %s", *launcherPid)
		netns, err := ns.GetNS(fmt.Sprintf("/proc/%s/ns/net", *launcherPid))

		if err != nil {
			glog.Fatalf("Could not load netns: %+v", err)
		} else if netns != nil {
			glog.V(4).Info("Successfully loaded netns ...")

			err = netns.Do(func(_ ns.NetNS) error {
				if err := createTapDevice(*tapName, uint(*uid), uint(*gid), false); err != nil {
					glog.Fatalf("error creating tap device: %v", err)
				}

				glog.V(4).Infof("Managed to create the tap device in pid %s", *launcherPid)
				return nil
			})
		}
	} else {
		if err := createTapDevice(*tapName, uint(*uid), uint(*gid), false); err != nil {
			glog.Fatalf("error creating tap device: %v", err)
		}
	}
	glog.V(4).Info("All good in tha hood")
	glog.Flush()
}
