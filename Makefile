default:
	go build -o cmd/

windows:
	GOOS=windows go build -o cmd/
	strip -s cmd/

linux:
	GOOS=linux go build -o cmd/

run:
	go run main.go
