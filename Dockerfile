# Dockerfile

# Use the official Go image as a parent image
FROM golang:1.21-alpine

# Set the working directory
WORKDIR /app

# Copy the local package files to the container's workspace
ADD . /app

# Build the Go app
RUN go build -o myhttpserver .

# Command to run the executable
CMD ["./myhttpserver"]
