ucloud: 
	CGO_ENABLED=0 GOOS=linux go build -mod vendor --ldflags "-s" -o bin/vm-controller
