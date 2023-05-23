package main

import (
	"fmt"
	"net"
	"net/http"
)

//start simple with just file server and uploader
//how to go about finding other machines in the network
//serve a single page that can upload to all machines or selective ones
//figure out how to do it on mac vs pc

const DEFAULTPORT = 7122

func main() {
	fmt.Println("Let's serve our files to our graphics machines")

	//scanline for computer name

	//

	//Routes
	// "/"

	// listen and serve
	thePort := fmt.Sprintf(":%v", DEFAULTPORT)
	theRootPath := "/"
	theFileSystem := http.Dir(theRootPath)
	theFileServer := http.FileServer(theFileSystem)
	go http.ListenAndServe(thePort, theFileServer)
	myIP := getMyLocalIP()
	fmt.Printf("your local files are being served at %v:%v \n*********the \"ENTER\" key will kill the server**********", myIP, DEFAULTPORT)

	//open the browser

	//need a pause here
	fmt.Scanln()

	fmt.Println("end of line.......................................")
}

func getMyLocalIP() net.IP {
	var myIP net.IP
	tt, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	for _, t := range tt {
		aa, err := t.Addrs()
		if err != nil {
			panic(err)
		}
		for _, a := range aa {
			ipnet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			v4 := ipnet.IP.To4()
			if v4 == nil || v4[0] == 127 { // loopback address
				continue
			}
			//fmt.Printf("%v\n", v4)
			myIP = v4
		}

	}
	return myIP
}
