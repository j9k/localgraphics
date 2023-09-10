hello:
	echo "Lets build and copy for all of the potential oss and hardware"



build:
	
	GOOS=darwin GOARCH=amd64    go build -o deploy/macintel/localGFXintel main.go

	GOOS=darwin GOARCH=arm64  go build -o deploy/macarm/localGFXarm main.go

	GOOS=windows GOARCH=amd64 go build -o deploy/windows/localGFXwin.exe main.go
