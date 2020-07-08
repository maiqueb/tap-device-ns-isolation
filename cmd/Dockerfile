FROM golang:1.13 as builder
ENV GOPATH=/go
WORKDIR /go/src/github.com/maiqueb/tap-experiments
COPY . .
RUN GOOS=linux CGO_ENABLED=e go build -o /tap-experiments github.com/maiqueb/tap-experiments/cmd

FROM registry.access.redhat.com/ubi8/ubi-minimal
COPY --from=builder /tap-experiments /tap-experiments
RUN microdnf install -y iproute
