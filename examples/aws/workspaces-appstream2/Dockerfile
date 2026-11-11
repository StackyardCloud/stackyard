FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /workspaces-appstream2 ./examples/aws/workspaces-appstream2
ENTRYPOINT ["/workspaces-appstream2"]
