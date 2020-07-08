# tap-device-ns-isolation
Experiments about creating &amp; consuming tap devices in different namespaces

## Build the docker container
```bash
make docker-build
```

## Start a privileged container - must share the host pid ns
```bash
docker run -ti --rm --privileged --pid=host --name create-tap tap-experiment bash
```

## Start a regular container
```bash
docker run -ti --rm --name consume-tap tap-experiment bash
```

