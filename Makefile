all:
	go test ./...
	go test ./... -short -race
	go test ./... -run=NONE -bench=. -benchmem
