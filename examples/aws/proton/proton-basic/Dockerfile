FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /proton-basic ./examples/aws/proton/proton-basic
ENTRYPOINT ["/proton-basic"]
