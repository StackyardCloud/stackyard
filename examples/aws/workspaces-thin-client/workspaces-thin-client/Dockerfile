FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /workspaces-thin-client ./examples/aws/workspaces-thin-client/workspaces-thin-client
ENTRYPOINT ["/workspaces-thin-client"]
