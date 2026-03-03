FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /neptunedata-basic ./examples/aws/neptunedata/neptunedata-basic
ENTRYPOINT ["/neptunedata-basic"]
