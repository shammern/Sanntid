go run main.go -id=1 -role=master -listen=127.0.0.1:8001 -peers=127.0.0.1:8002,127.0.0.1:8003
go run main.go -id=2 -role=backup -listen=127.0.0.1:8002 -peers=127.0.0.1:8001,127.0.0.1:8003
go run main.go -id=3 -role=idle -listen=127.0.0.1:8003 -peers=127.0.0.1:8001,127.0.0.1:8002
