# ...existing code...

# Download dependencies first
RUN go mod download

# Verify dependencies
RUN go mod verify

# List directory structure for debugging
RUN ls -la cmd/ || echo "cmd directory not found"
RUN ls -la cmd/main.go || echo "cmd/main.go not found"

# Build the application with JupyterHub integration
RUN CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -o main cmd/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -o init cmd/init/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -o test-k8s cmd/test-k8s/main.go

# ...existing code...

