FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /comprehend-medical-basic ./examples/comprehend-medical/comprehend-medical-basic
ENTRYPOINT ["/comprehend-medical-basic"]
