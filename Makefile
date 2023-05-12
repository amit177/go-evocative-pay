default:
	go build -ldflags="-s -w" -o cmd/

windows:
	GOOS=windows go build -ldflags="-s -w" -o cmd/

linux:
	GOOS=linux go build -ldflags="-s -w" -o cmd/

run:
	go run main.go
