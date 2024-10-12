# Use an official Golang runtime as the base image
FROM golang:1.20-alpine

# Set the working directory inside the container
WORKDIR /app

# Copy the Go source files to the container
COPY . .

# Build the Go server
RUN go mod tidy && go build -o go-calculator server/calculator/main.go 

# Expose the port (default is 8080, but you can change it when running the container)
EXPOSE 9090

# Run the server (can override port with env variable)
CMD ["./go-calculator", "-port=8080"]
