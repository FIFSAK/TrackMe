FROM golang:1.23-alpine as builder

WORKDIR /build

COPY . /build

# Build the Go application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o TrackMe-service .

# Create a new stage for the final application image
FROM alpine:3.18 as hoster

# Copy the built application from the builder stage
COPY --from=builder /build/.env ./.env
COPY --from=builder /build/TrackMe-service ./TrackMe-service
COPY --from=builder /build/stages.yaml ./stages.yaml

EXPOSE 80

# Define the entry point for the final application image
ENTRYPOINT ["./TrackMe-service"]