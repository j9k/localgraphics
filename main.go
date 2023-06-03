package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
)

//start simple with just file server and uploader
//how to go about finding other machines in the network
//serve a single page that can upload to all machines or selective ones
//figure out how to do it on mac vs pc

const DEFAULTPORT = 7122

type server struct {
	Port         int
	Name         string
	HostName     string
	IPAddress    net.IP
	RootPath     string
	HTMLTemplate *template.Template
}

func main() {
	fmt.Println("Let's serve our files to our graphics machines")
	//collect all the data to serve the web page

	hst, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	var theServer server = server{
		Port:         DEFAULTPORT,
		Name:         "",
		HostName:     hst,
		IPAddress:    getMyLocalIP(),
		RootPath:     "/",
		HTMLTemplate: GiveMeUploadHTMLPointer(),
	}

	//scanline for computer name

	//
	thePort := fmt.Sprintf(":%v", theServer.Port)

	theFileSystem := http.Dir(theServer.RootPath)
	theFileServer := http.FileServer(theFileSystem)
	//Routes
	// "/"
	mux := http.NewServeMux()
	mux.Handle("/", theFileServer)
	mux.HandleFunc("/upload", theServer.uploadPageHandler) //serves the file
	mux.HandleFunc("/upl", theUploadHandler)               //endpoint for file uploads
	//mux.NotFoundHandler(redirectToRoot)
	//http.Handle("/", mux)

	// listen and serve

	go http.ListenAndServe(thePort, mux)
	myIP := getMyLocalIP()
	fmt.Printf("your local files are being served at %v:%v \n*********the \"ENTER\" key will kill the server**********\n", myIP, DEFAULTPORT)

	//open the browser

	//need a pause here
	fmt.Scanln()

	fmt.Println("end of line.......................................")
}

//*********************************************************************************************

// turn this into a method with a pointer to the html template
func (serv *server) uploadPageHandler(rw http.ResponseWriter, r *http.Request) {
	//rw.Header().Add("Content-Type", "text/html")
	//http.ServeFile(rw, r, "index.html")
	err := serv.HTMLTemplate.ExecuteTemplate(rw, "upload", nil)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}
}

// this is the destination of the uploaded file
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

	dst, err := os.Create(fmt.Sprintf("./uploads/%s", fileHeader.Filename))
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

// need syntax for html
func GiveMeUploadHTMLPointer() *template.Template {

	const uploadPageConst = ` {{define "upload"}}
<!DOCTYPE html>
<html>
 <body>
<form
  id="form"
  enctype="multipart/form-data"
  action="/upl"
  method="POST"
>
  <input class="input file-input" type="file" name="file" multiple />
  <button class="button" type="submit">Submit</button>
</form>
<p>UPLOAD YOUR FILES</p>
</body>

</html>
{{end}}`

	theUploadTemplatePointer, err := template.New("upload").Parse(uploadPageConst)
	if err != nil {
		log.Fatal(err)
	}
	return theUploadTemplatePointer
}

//func redirectToRoot(w http.ResponseWriter, r *http.Request) {
//	http.Redirect(w, r, "/", http.StatusSeeOther)
//}
