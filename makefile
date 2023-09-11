hello:
	echo "Lets build and copy for all of the potential oss and hardware"

delete:
	rm deploy/macintel/localGFXintel
	rm deploy/macarm/localGFXarm
	rm deploy/windows/localGFXwin.exe


build:
	
	

	GOOS=darwin GOARCH=amd64    go build -o deploy/macintel/localGFXintel main.go


	GOOS=darwin GOARCH=arm64  go build -o deploy/macarm/localGFXarm main.go


	GOOS=windows GOARCH=amd64 go build -o deploy/windows/localGFXwin.exe main.go
 	zip -r localGFX.zip deploy