test:
	go test -v

cover:
	rm -rf *.coverprofile
	go test -v -coverprofile=token-bucket.coverprofile
	gover
	go tool cover -html=token-bucket.coverprofile