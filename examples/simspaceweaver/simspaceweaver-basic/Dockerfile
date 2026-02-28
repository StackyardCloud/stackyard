FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /simspaceweaver-basic ./examples/simspaceweaver/simspaceweaver-basic
ENTRYPOINT ["/simspaceweaver-basic"]
