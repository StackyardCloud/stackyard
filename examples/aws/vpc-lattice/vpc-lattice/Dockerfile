FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /vpc-lattice ./examples/aws/vpc-lattice/vpc-lattice
ENTRYPOINT ["/vpc-lattice"]
