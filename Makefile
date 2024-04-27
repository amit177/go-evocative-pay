default:
	go build -trimpath -ldflags="-s -w" -o cmd/

windows:
	GOOS=windows go build -trimpath -ldflags="-s -w" -o cmd/

linux:
	GOOS=linux go build -trimpath -ldflags="-s -w" -o cmd/

run:
	go run main.go

clean:
	rm -fv cmd/*
