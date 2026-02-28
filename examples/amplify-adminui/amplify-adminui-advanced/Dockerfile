FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /amplify-adminui-advanced ./examples/amplify-adminui/amplify-adminui-advanced
ENTRYPOINT ["/amplify-adminui-advanced"]
