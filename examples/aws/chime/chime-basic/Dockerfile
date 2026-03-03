FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /chime-basic ./examples/aws/chime/chime-basic
ENTRYPOINT ["/chime-basic"]
