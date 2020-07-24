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

## Deploy into a running kubevirtci based cluster
Use the following instructions to deploy the sandbox containers into an
existing KubeVirt cluster.

This document assumes the cluster has been properly deployed, and shell that
executes it has kubeconfig properly set.

### Build the container / push it to the registry
These commands are executed on your local development environment. They will
build the container image with the executable, and push it to your k8s cluster
registry.

```bash
push_registry=localhost:$("$PATH_TO_KUBEVIRT_SRC_CODE"/cluster-up/cli.sh ports registry | tr -d '\r')
IMAGE_REGISTRY=$push_registry make docker-build
IMAGE_REGISTRY=$push_registry make docker-push
```

### Pull the image on the k8s node
This command will download the correct image from your k8s cluster registry
into the docker registry of the k8s node we will interact with in the future.

It must be executed from within the desired k8s node.

```bash
docker pull registry:5000/tap-experiments
```

### Create the containers on the k8s node
These following commands should be executed from the k8s node.
They create 2 containers (tap creator + tap consumer), and provision an selinux
policy that explicitly grants a set of permissions to consume the tun socket.

```bash
# create the privileged container - must mount another volume, to make the
# selinux policy file available to the k8s node
$ docker run -ti --rm --privileged --pid=host --name create-tap \
  -v /dev/net/tun:/dev/net/tun \
  -v /root/selinux-policies:/selinux-policies/ \
  registry:5000/tap-experiment:latest \
  bash

# create the consumer container
$ docker run -ti --rm --name consume-tap \
  -v /dev/net/tun:/dev/net/tun \
  registry:5000/tap-experiment:latest \
  bash

# provision the selinux policy
$ docker exec create-tap /tap-maker exec --mount /proc/1/ns/mnt exec -- \
    /usr/sbin/semodule -i /root/selinux-policies/allow_clone_dev_access.cil
```

