all:
	protoc --go_out=. *.proto
	go build
