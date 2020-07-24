#!/bin/bash -ex

node_name="${node:-node01}"
tap_name="${tap:-tap0}"

k8s_provider="${KUBEVIRT_PROVIDER:-k8s-1.18}"
kubevirt_src_code="${KUBEVIRT_CODE:-$GOROOT/src/github.com/kubernetes/kubevirt/}"
export KUBEVIRT_PROVIDER=$k8s_provider
export KUBECONFIG=$("$kubevirt_src_code"/cluster-up/kubeconfig.sh)

push_registry=localhost:$("$kubevirt_src_code"/cluster-up/cli.sh ports registry | tr -d '\r')
IMAGE_REGISTRY=$push_registry make docker-build
IMAGE_REGISTRY=$push_registry make docker-push

function run_in_node {
    local node_name="$1"
    local command="$2"
    "$kubevirt_src_code/cluster-up/ssh.sh" "$node_name" "$command"
}

run_in_node "$node_name" "sudo docker pull registry:5000/tap-experiment"

run_in_node "$node_name" "
    sudo docker run -ti -d --privileged --pid=host --name create-tap \
      -v /dev/net/tun:/dev/net/tun \
      -v /root/selinux-policies:/selinux-policies/ \
      registry:5000/tap-experiment:latest \
      bash
"

run_in_node "$node_name" "
    sudo docker run -ti -d --name consume-tap \
      -v /dev/net/tun:/dev/net/tun \
      registry:5000/tap-experiment:latest \
      bash
"

run_in_node "$node_name" "
    sudo docker exec create-tap \
      cp /allow_clone_dev_access.cil \
        /selinux-policies/allow_clone_dev_access.cil
"

run_in_node "$node_name" "
    sudo docker exec create-tap \
      /tap-maker exec --mount /proc/1/ns/mnt -- \
        /usr/sbin/semodule -i /root/selinux-policies/allow_clone_dev_access.cil
"

consumer_pid=$(
    run_in_node "$node_name" "
        sudo docker inspect consume-tap \
          -f '{{ .State.Pid }}'
    " | tail -n1 | tr -d '\r'
)

run_in_node "$node_name" "
    sudo docker exec create-tap \
      /tap-maker create-tap --tap-name $tap_name -p $consumer_pid -v 9
"
