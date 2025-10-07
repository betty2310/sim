```bash
git clone https://github.com/betty2310/sim.git
go mod download
go mod tidy

# Run the application
go run cmd/simulator/main.go

# Or build and run
go build -o device-simulator cmd/simulator/main.go
./device-simulator
```
