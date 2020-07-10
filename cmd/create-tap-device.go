package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/golang/glog"
	"github.com/songgao/water"
)

func setupLogger() {
	if err := flag.Set("logtostderr", "true"); err != nil {
		os.Exit(1)
	}
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
			if err := createTapDevice(tapName, uid, gid, false); err != nil {
				glog.Fatalf("error creating tap device: %v", err)
			}

			glog.V(4).Infof("Managed to create the tap device in pid %s", launcherPid)
			return nil
		})
	}
}

func main() {
	setupLogger()

	tapName := flag.String("tap-name", "tap0", "override the name of the tap device")
	launcherPid := flag.String("launcher-pid", "", "optionally specify the PID holding the netns where the tap device will be created.")
	serveStuff := flag.Bool("consume-tap", false, "Indicate that this process is meant to just sit there and consume the tap device")
	uidInput := flag.Int("uid", 0, "the owner UID of the tap device")
	gidInput := flag.Int("gid", 0, "the owner GID of the tap device")

	flag.Parse()
	appMode := "create-tap"
	uid := uint(*uidInput)
	gid := uint(*gidInput)

	if *serveStuff {
		appMode = "consume-tap"
		glog.V(4).Infof("Started app in %s mode", appMode)
		err := createTapDevice(*tapName, uid, gid, false)
		if err != nil {
			glog.Fatalf("Could not open the tapsy-thingy: %+v", err)
		}

		glog.V(4).Infof("Opened the tap device on pid %d", os.Getpid())
		for {
			time.Sleep(time.Second)
		}
	}

	glog.V(4).Infof("Started app in %s mode", appMode)
	if *launcherPid != "" {
		glog.V(4).Infof("Executing in netns of pid %s", *launcherPid)
		createTapDeviceOnPIDNetNs(*launcherPid, *tapName, uid , gid)
	} else {
		if err := createTapDevice(*tapName, uid, gid, false); err != nil {
			glog.Fatalf("error creating tap device: %v", err)
		}
	}

	glog.Flush()
}
