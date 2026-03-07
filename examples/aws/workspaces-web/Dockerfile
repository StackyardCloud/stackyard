FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /workspaces-web ./examples/aws/workspaces-web
ENTRYPOINT ["/workspaces-web"]
