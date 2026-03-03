FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /workspaces-thin-client-advanced ./examples/aws/workspaces-thin-client/workspaces-thin-client-advanced
ENTRYPOINT ["/workspaces-thin-client-advanced"]
