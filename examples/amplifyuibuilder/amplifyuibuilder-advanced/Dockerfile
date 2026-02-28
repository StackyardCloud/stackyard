FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /amplifyuibuilder-advanced ./examples/amplifyuibuilder/amplifyuibuilder-advanced
ENTRYPOINT ["/amplifyuibuilder-advanced"]
