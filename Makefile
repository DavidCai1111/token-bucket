test:
	go test -v -race

cover:
	rm -rf *.coverprofile
	go test -v -coverprofile=token-bucket.coverprofile
	gover
	go tool cover -html=token-bucket.coverprofile
	rm -rf *.coverprofile
