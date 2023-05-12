default:
	go build -o cmd/

windows:
	GOOS=windows go build -o cmd/evocativepay.exe
	strip -s cmd/

linux:
	GOOS=linux go build -o cmd/evoactivepay

run:
	go run main.go
