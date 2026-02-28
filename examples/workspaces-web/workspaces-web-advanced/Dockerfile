FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /workspaces-web-advanced ./examples/workspaces-web/workspaces-web-advanced
ENTRYPOINT ["/workspaces-web-advanced"]
