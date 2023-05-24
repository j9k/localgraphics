package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
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
	thePort := fmt.Sprintf(":%v", DEFAULTPORT)
	theRootPath := "/"
	theFileSystem := http.Dir(theRootPath)
	theFileServer := http.FileServer(theFileSystem)
	//Routes
	// "/"
	mux := http.NewServeMux()
	mux.Handle("/", theFileServer)
	mux.HandleFunc("/upload", uploadPageHandler) //serves the file
	mux.HandleFunc("/upl", theUploadHandler)     //endpoint for file uploads
	// listen and serve

	go http.ListenAndServe(thePort, mux)
	myIP := getMyLocalIP()
	fmt.Printf("your local files are being served at %v:%v \n*********the \"ENTER\" key will kill the server**********", myIP, DEFAULTPORT)

	//open the browser

	//need a pause here
	fmt.Scanln()

	fmt.Println("end of line.......................................")
}

func uploadPageHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "text/html")
	http.ServeFile(rw, r, "index.html")
}

func theUploadHandler(rw http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	defer file.Close()

	// Create the uploads folder if it doesn't
	// already exist
	err = os.MkdirAll("./uploads", os.ModePerm)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create a new file in the uploads directory
	var fileNameString string

	fileNameString = fileHeader.Filename // This line in particular is what you're looking for.

	dst, err := os.Create(fmt.Sprintf("./uploads/%s", fileNameString))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	defer dst.Close()

	// Copy the uploaded file to the filesystem
	// at the specified destination
	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(rw, "Upload successful")
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
