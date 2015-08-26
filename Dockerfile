# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang:1.4.2

# Copy the local package files to the container's workspace.
ADD . /go/src/github.com/yhat/workload-simulator


# Build the outyet command inside the container.
# (You may fetch or manage dependencies here,
# either manually or with a tool like "godep".)
RUN go get github.com/tools/godep
WORKDIR /go/src/github.com/yhat/workload-simulator
RUN godep go install -v ./...
RUN mkdir -p /root/workload_sim

# Run the outyet command by default when the container starts.
ENTRYPOINT workload-simulator -config workload-simulator.yaml

# Document that the service listens on port 8080.
EXPOSE 8080
