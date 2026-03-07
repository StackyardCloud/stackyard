FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /vpc-lattice ./examples/aws/vpc-lattice
ENTRYPOINT ["/vpc-lattice"]
