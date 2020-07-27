FROM golang:1.13 as builder
ENV GOPATH=/go
WORKDIR /go/src/github.com/maiqueb/tap-experiments
COPY . .
RUN GOOS=linux CGO_ENABLED=e go build -o /tap-experiments -tags=selinux github.com/maiqueb/tap-experiments/cmd

FROM registry.access.redhat.com/ubi8/ubi-minimal
COPY --from=builder /tap-experiments /tap-maker
RUN microdnf install -y iproute procps-ng

ADD allow_clone_dev_access.cil /allow_clone_dev_access.cil
RUN mkdir -p /selinux-policies && \
        cp /allow_clone_dev_access.cil /selinux-policies
