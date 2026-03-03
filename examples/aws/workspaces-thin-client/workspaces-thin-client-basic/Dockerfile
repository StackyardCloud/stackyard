FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /workspaces-thin-client-basic ./examples/aws/workspaces-thin-client/workspaces-thin-client-basic
ENTRYPOINT ["/workspaces-thin-client-basic"]
