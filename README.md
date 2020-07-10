# tap-device-ns-isolation
Experiments about creating &amp; consuming tap devices in different namespaces

## Build the docker container
```bash
make docker-build
```

## Start a privileged container - must share the host pid ns, and volume mount the clone device
```bash
docker run -ti --rm --privileged --pid=host --name create-tap -v /dev/net/tun:/dev/net/tun tap-experiment bash
```

## Start a regular container - needs to have the clone device volume mounted into it
```bash
docker run -ti --rm --name consume-tap -v /dev/net/tun:/dev/net/tun tap-experiment bash
```

## Figure out the pid of the un-privileged container
```bash
launcher_pid=$(docker inspect consume-tap -f '{{ .State.Pid }}')
```

## Create a tap device on the target process PID (on the privileged container)
```bash
/tap-maker create-tap --tap-name tap25 --launcher-pid <launcher_pid>
```

## Connect to the created tap device (on the un-privileged container)
```bash
/tap-maker consume-tap --tap-name tap25
```

