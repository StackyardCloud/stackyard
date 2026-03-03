FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /workspaces-appstream2 ./examples/aws/workspaces-appstream2/workspaces-appstream2
ENTRYPOINT ["/workspaces-appstream2"]
