FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /simspaceweaver-advanced ./examples/simspaceweaver/simspaceweaver-advanced
ENTRYPOINT ["/simspaceweaver-advanced"]
