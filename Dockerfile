FROM golang AS builder

# Set destination for COPY
WORKDIR /source

# Download Go modules
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/reference/dockerfile/#copy
COPY ./cmd ./
COPY ./pkg ./

# Add the -ldflags '-w -s' flags to reduce the size of the binary
RUN CGO_ENABLED=0 go build -a -ldflags '-w -s' -o /app/bot /source/main.go

# Now copy it into a base image.
FROM alpine

# Create a group and user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Tell docker that all future commands should run as the appuser user
USER appuser

COPY --from=builder /app/bot /app/bot
CMD ["/app/bot"]