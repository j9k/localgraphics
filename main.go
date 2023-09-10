package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/skip2/go-qrcode"
)

//start simple with just file server and uploader
//how to go about finding other machines in the network
//serve a single page that can upload to all machines or selective ones
//figure out how to do it on mac vs pc

var computerName string
var portFlag int
var templates map[string]*template.Template
var err error
var serverStore string = "/uploads"
var upgrader = websocket.Upgrader{}
var WebPage string

func init() {
	flag.StringVar(&computerName, "cn", "name not set", "The Role of or name of the computer GFX-1, Playback..")
	flag.IntVar(&portFlag, "port", 7122, "The port you want the server to use. default is 7122")

	if templates == nil {
		templates = make(map[string]*template.Template)
	}
	templates["index"], err = template.New("index").Parse(indexPageConst)
	ifErrFatal(err)
	templates["upload"], err = template.New("uploads").Parse(uploadPageConst)
	ifErrFatal(err)
	templates["options"], err = template.New("options").Parse(optionsPageConst)
}

type server struct {
	Port           int    `json:"port"`
	Name           string `json:"name"`
	HostName       string `json:"hostname"`
	IPAddress      net.IP `json:"ipaddress"`
	WD             fs.FS
	FilesPath      string `json:"filespath"`
	UploadPath     string `json:"uploadpath"`
	TemplatesMap   *map[string]*template.Template
	OtherComputers *[]OtherComputer `json:"othercomputers"`
	UploadQR       *string
	FilesQR        *string
}

type OtherComputer struct {
	Name string  `json:"name"`
	Link url.URL `json:"link"`
}

func NewOtherComputers() *[]OtherComputer {
	oc := make([]OtherComputer, 0, 50)
	return &oc
}
func main() {
	flag.Parse()
	fmt.Println("Let's serve our files to our graphics machines")
	//collect all the data to serve the web page

	hst, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	//lclIP := getMyLocalIP()

	// use flags to determine if elsesss
	uplPath := "/upload"

	wdString, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	wdFS := os.DirFS(wdString)

	var theServer server = server{
		Port:           portFlag,
		Name:           "This is Name",
		HostName:       hst,
		WD:             wdFS,
		IPAddress:      getMyLocalIP(),
		FilesPath:      "/",
		UploadPath:     uplPath,
		TemplatesMap:   &templates,
		OtherComputers: NewOtherComputers(), //default is 50 probably more than anone will ever need
		//	UploadQR:           new(string),
		//	FilesQR:            new(string),
	}

	fmt.Println("theServer", theServer)
	pToTheServ := &theServer

	thePortString := fmt.Sprintf(":%v", portFlag)
	pToTheServ.PrintServerStruct()
	//	theFileSystem := http.Dir(theServer.FilesPath)
	//	theFileServer := http.FileServer(theFileSystem)
	//Routes
	// "/"
	mux := http.NewServeMux()
	//	mux.Handle("/files", theFileServer)

	fs := http.FileServer(http.Dir("./"))
	mux.Handle("/files", fs)

	//mux.Handle("/f", http.FileServer(http.Dir("../")))

	mux.HandleFunc("/admin", pToTheServ.indexPageHandler)
	mux.HandleFunc("/options", pToTheServ.optionsHandler)
	mux.HandleFunc("/options/data", pToTheServ.optionsPostHandler)
	mux.HandleFunc("/upload", theUploadHandler) //serves the file
	mux.HandleFunc("/upl", theUploadHandler)    //endpoint for file uploads

	mux.HandleFunc("/", mooSocket)
	mux.HandleFunc("/ws", theWebsocket)

	// listen and serve

	MyIP := getMyLocalIP()
	fmt.Printf("your local files are being served at %v%v \n*********the \"ENTER\" key will kill the server**********\n", MyIP, thePortString)

	//open the browser

	WebPage = fmt.Sprintf("http://%s:%v", MyIP, portFlag)
	err = openWebPage(WebPage)
	fmt.Println(WebPage)
	if err != nil {
		log.Panicln(err)
	}

	http.ListenAndServe(thePortString, mux)

	//need a pause here
	//fmt.Scanln()

	fmt.Println("end of line.......................................")
}

// ********************        HANDLERS          *************************************************************************
func (serv *server) indexPageHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Println("top of indexPageHandler")
	renderTemplate(w, "index", "index", serv)
}

// want to be able to change the name and port of the server
func (serv *server) optionsHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Println("optionsHandler")
	type OptionsView struct {
		PageName     string
		IP           string
		ComputerName string
	}
	var opts OptionsView = OptionsView{
		PageName:     "Options",
		ComputerName: serv.Name,
	}
	computerName = serv.Name
	opts.IP = r.RemoteAddr

	renderTemplate(w, "options", "options", opts)
}

// take in the form data and modify the server
func (serv *server) optionsPostHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	asdf := r.Form
	fmt.Println(asdf)
	serv.Name = asdf.Get("cname")
	http.Redirect(w, r, "/options", http.StatusFound)
}

// this is the destination of the uploaded file
func theUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file, fileHeader, err := r.FormFile("filenameformname")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	defer file.Close()

	// Create the uploads folder if it doesn't
	// already exist
//	err = os.MkdirAll("./uploads", os.ModePerm)
//	if err != nil {
//		http.Error(w, err.Error(), http.StatusInternalServerError)
//		return
//	}

	// Create a new file in the uploads directory

	dst, err := os.Create(fmt.Sprintf("./%s", fileHeader.Filename))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer dst.Close()

	// Copy the uploaded file to the filesystem
	// at the specified destination
	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Upload successful")
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

// ********************** TEMPLATE FUNCTIONS *************************************
// renders the template with the data
func renderTemplate(w http.ResponseWriter, name string, template string, viewModel interface{}) {
	tmpl, ok := templates[name]
	if !ok {
		http.Error(w, "the template does not exist", http.StatusInternalServerError)
	}
	err := tmpl.ExecuteTemplate(w, template, viewModel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// ****************** QR FUNCTIONS *****************************************

func (serv *server) GiveMeFilesQRpointer() *string {
	var port string = fmt.Sprintf(":%v", serv.Port)

	if serv.Port == 80 {
		port = ""
	}

	data := fmt.Sprintf("http://%s%s/%s", serv.IPAddress, port, serv.FilesPath)

	return ComposeQRData(data)
}

func (serv *server) GiveMeUploadQRpointer() *string {
	var port string = fmt.Sprintf(":%v", serv.Port)

	if serv.Port == 80 {
		port = ""
	}

	data := fmt.Sprintf("http://%s%s/%s", serv.IPAddress, port, serv.UploadPath)
	fmt.Println(data)
	return ComposeQRData(data)
}

func ComposeQRData(data string) (qr *string) {
	size := 256

	//fileName := fmt.Sprintf("%sQR%v.png", site, size)

	//qrFullString := fmt.Sprintf("http://%s", data)
	//	err := qrcode.WriteFile(qrFullString, qrcode.Highest, size, fileName)
	//	if err != nil {
	//		log.Panicln(err)
	//	}
	//generate the byte slice
	pngQR, err := qrcode.Encode(data, qrcode.Highest, size)
	if err != nil {
		log.Panicln(err)
	}

	//encode to base64
	b64qr := base64.RawStdEncoding.EncodeToString(pngQR)

	//      <img src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAUAAAAFCAYAAACNbyblAAAAHElEQVQI12P4
	//8/w38GIAXDIBKE0DHxgljNBAAO9TXL0Y4OHwAAAABJRU5ErkJggg==" alt="Red dot" />

	//create the string for the website
	//gotta get all those escape characters in there
	htmlQR := fmt.Sprintf("<img src=\"data:image/png;base64,%s\" alt=\"QR\"/>", b64qr)

	//fmt.Println(htmlQR)

	//if err != nil {
	//	log.Println(err)
	//}
	qr = &htmlQR
	return qr
}

// ******************************** Utility ****************************************
func (serv *server) PrintServerStruct() {
	fmt.Printf("Port === %v\n", serv.Port)
	fmt.Printf("FilesPath === %v\n", serv.FilesPath)
	fmt.Printf("UploadPath === %v\n", serv.UploadPath)
	fmt.Printf("IP Address === %v\n", serv.IPAddress)
	fmt.Printf("HostName === %v\n", serv.HostName)
	fmt.Printf("Name === %v\n", serv.Name)
}

func ifErrFatal(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func (serv *server) FindOtherComputers() {
	//need to set up a client and scrape
}

// **********************************  HTML   ********************************************************
const uploadPageConst = `{{define "Upload"}}
<!DOCTYPE html>
<html>
 <body>
<form
  id="form"
  enctype="multipart/form-data"
  action="/upl"
  method="POST">
  <input class="input file-input" type="file" name="file" multiple />
  <button class="button" type="submit">Submit</button>
</form>
<p>UPLOAD YOUR FILES</p>
</body>

</html>
{{end}}`

const indexPageConst = ` {{define "index"}}
<!DOCTYPE html>
<html>
<body>
<h1>Local File Server</h1>
<h2>{{.Name}}</h2>
<h2>{{.IPAddress}}:{{.Port}}</h2>
<h2>Working Directory {{.WD}}</h2>
<h2>Upload path {{.UploadPath}}</h2>
<a href="/options">Options</a>
<a href="/upload">Upload</a>
<a href="/files">Files</a>

<p>
<a href="/">websocket upload</a>
</p>


<h1>Files</h1>

</body>

</html>
{{end}}`

const optionsPageConst = `{{define "options"}}
<!DOCTYPE html>
<html>
<body>
<h1>{{.PageName}}</h1>
<h2>{{.IP}}</h2>


<p>Current Computer Name: {{.ComputerName}} </p>

<form action="/options/data">
  <label for="compname">Computer Name:</label>
  <input type="text" id="cname" name="cname"><br><br>
  <input type="submit" value="Submit">
</form>

<form action="/options/data">
  <label for="compname">Other Computers:</label>
  <input type="text" id="othername" name="othername"><br><br>

  <label for="otherip">Other Computers IP:</label>
  <input type="text" id="otherip" name="otherip"><br><br>
  <input type="submit" value="Submit">



</form>


<p>
<a href="/admin">Back</a>
</p>
</body>
<h1>moooo</h1>



 
</body>

</html>
{{end}}
`

// need to range this
const otherComputers = `{{define "OtherComputers"}}


<a href="http://{{.OtherIP}}">Back</a>

{{end}}
`

/*
<p>
<img src=\"data:image/png;base64,{{.FilesQR}}\" alt=\"QR\"/>
</p>
<h1>Upload</h1>
<p>
<img src=\"data:image/png;base64,{{.UploadQR}}\" alt=\"QR\"/>
</p>
 <body>
*/

func openWebPage(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
		args = append(args, url)
		exec.Command(cmd, args...).Start()
	case "darwin": //mac
		cmd = "open"
		args = append(args, url)
		exec.Command(cmd, args...).Start()
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = ""
	}

	return nil
}

//Websocket uploader **************************************

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func theWebsocket(w http.ResponseWriter, r *http.Request) {
	var err error
	// not safe, only for dev:
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("error", err)
		return
	}
	log.Println("connection", r.RemoteAddr)
	filename := ""
	var f *os.File

	go func(conn *websocket.Conn) {
		fmt.Println(os.Getwd())
		err := os.Chdir("uploads") //don't use forward slash
		if err != nil {
			log.Println(err)
		}

		for {

			mt, data, connErr := conn.ReadMessage()
			if connErr != nil {
				log.Println("error", connErr)
				return
			}
			if mt == 1 {
				event := strings.Split(string(data), ":")
				if event[0] == "upload" {
					filename = event[2] // 1 is the md5 2 is the file name string
					if fileExists(filename) {
						log.Println(filename + " already exists")
						if err := conn.WriteMessage(1, []byte("exists")); err != nil {
							log.Println("error sending exists message")
						}
					} else {
						f, err = os.Create(filename)
						if err != nil {
							log.Println(err)
						}
					}
				}
				log.Println(string(event[0]), filename)
				if event[0] == "ready" {
					os.Chdir("./")
					f.Close()

					if err := conn.WriteMessage(1, []byte("ready")); err != nil {
						log.Println("error sending ready message")
					}
					filename = ""
				}
			}
			if mt == 2 {
				log.Println("chunk", filename)

				f.Write(data)
			}
		}
	}(conn)
}

/*
func mimeType(filename string) string {
	if f, err := os.Open(filename); err == nil {
		buffer := make([]byte, 512)
		if _, err := f.Read(buffer); err == nil {
			return http.DetectContentType(buffer)
		}
	}
	return ""
}
*/

func jsmd5() string {
	return `(function(factory){if(typeof exports==="object"){module.exports=factory()}else if(typeof define==="function"&&define.amd){define(factory)}else{var glob;try{glob=window}catch(e){glob=self}glob.SparkMD5=factory()}})(function(undefined){"use strict";var add32=function(a,b){return a+b&4294967295},hex_chr=["0","1","2","3","4","5","6","7","8","9","a","b","c","d","e","f"];function cmn(q,a,b,x,s,t){a=add32(add32(a,q),add32(x,t));return add32(a<<s|a>>>32-s,b)}function md5cycle(x,k){var a=x[0],b=x[1],c=x[2],d=x[3];a+=(b&c|~b&d)+k[0]-680876936|0;a=(a<<7|a>>>25)+b|0;d+=(a&b|~a&c)+k[1]-389564586|0;d=(d<<12|d>>>20)+a|0;c+=(d&a|~d&b)+k[2]+606105819|0;c=(c<<17|c>>>15)+d|0;b+=(c&d|~c&a)+k[3]-1044525330|0;b=(b<<22|b>>>10)+c|0;a+=(b&c|~b&d)+k[4]-176418897|0;a=(a<<7|a>>>25)+b|0;d+=(a&b|~a&c)+k[5]+1200080426|0;d=(d<<12|d>>>20)+a|0;c+=(d&a|~d&b)+k[6]-1473231341|0;c=(c<<17|c>>>15)+d|0;b+=(c&d|~c&a)+k[7]-45705983|0;b=(b<<22|b>>>10)+c|0;a+=(b&c|~b&d)+k[8]+1770035416|0;a=(a<<7|a>>>25)+b|0;d+=(a&b|~a&c)+k[9]-1958414417|0;d=(d<<12|d>>>20)+a|0;c+=(d&a|~d&b)+k[10]-42063|0;c=(c<<17|c>>>15)+d|0;b+=(c&d|~c&a)+k[11]-1990404162|0;b=(b<<22|b>>>10)+c|0;a+=(b&c|~b&d)+k[12]+1804603682|0;a=(a<<7|a>>>25)+b|0;d+=(a&b|~a&c)+k[13]-40341101|0;d=(d<<12|d>>>20)+a|0;c+=(d&a|~d&b)+k[14]-1502002290|0;c=(c<<17|c>>>15)+d|0;b+=(c&d|~c&a)+k[15]+1236535329|0;b=(b<<22|b>>>10)+c|0;a+=(b&d|c&~d)+k[1]-165796510|0;a=(a<<5|a>>>27)+b|0;d+=(a&c|b&~c)+k[6]-1069501632|0;d=(d<<9|d>>>23)+a|0;c+=(d&b|a&~b)+k[11]+643717713|0;c=(c<<14|c>>>18)+d|0;b+=(c&a|d&~a)+k[0]-373897302|0;b=(b<<20|b>>>12)+c|0;a+=(b&d|c&~d)+k[5]-701558691|0;a=(a<<5|a>>>27)+b|0;d+=(a&c|b&~c)+k[10]+38016083|0;d=(d<<9|d>>>23)+a|0;c+=(d&b|a&~b)+k[15]-660478335|0;c=(c<<14|c>>>18)+d|0;b+=(c&a|d&~a)+k[4]-405537848|0;b=(b<<20|b>>>12)+c|0;a+=(b&d|c&~d)+k[9]+568446438|0;a=(a<<5|a>>>27)+b|0;d+=(a&c|b&~c)+k[14]-1019803690|0;d=(d<<9|d>>>23)+a|0;c+=(d&b|a&~b)+k[3]-187363961|0;c=(c<<14|c>>>18)+d|0;b+=(c&a|d&~a)+k[8]+1163531501|0;b=(b<<20|b>>>12)+c|0;a+=(b&d|c&~d)+k[13]-1444681467|0;a=(a<<5|a>>>27)+b|0;d+=(a&c|b&~c)+k[2]-51403784|0;d=(d<<9|d>>>23)+a|0;c+=(d&b|a&~b)+k[7]+1735328473|0;c=(c<<14|c>>>18)+d|0;b+=(c&a|d&~a)+k[12]-1926607734|0;b=(b<<20|b>>>12)+c|0;a+=(b^c^d)+k[5]-378558|0;a=(a<<4|a>>>28)+b|0;d+=(a^b^c)+k[8]-2022574463|0;d=(d<<11|d>>>21)+a|0;c+=(d^a^b)+k[11]+1839030562|0;c=(c<<16|c>>>16)+d|0;b+=(c^d^a)+k[14]-35309556|0;b=(b<<23|b>>>9)+c|0;a+=(b^c^d)+k[1]-1530992060|0;a=(a<<4|a>>>28)+b|0;d+=(a^b^c)+k[4]+1272893353|0;d=(d<<11|d>>>21)+a|0;c+=(d^a^b)+k[7]-155497632|0;c=(c<<16|c>>>16)+d|0;b+=(c^d^a)+k[10]-1094730640|0;b=(b<<23|b>>>9)+c|0;a+=(b^c^d)+k[13]+681279174|0;a=(a<<4|a>>>28)+b|0;d+=(a^b^c)+k[0]-358537222|0;d=(d<<11|d>>>21)+a|0;c+=(d^a^b)+k[3]-722521979|0;c=(c<<16|c>>>16)+d|0;b+=(c^d^a)+k[6]+76029189|0;b=(b<<23|b>>>9)+c|0;a+=(b^c^d)+k[9]-640364487|0;a=(a<<4|a>>>28)+b|0;d+=(a^b^c)+k[12]-421815835|0;d=(d<<11|d>>>21)+a|0;c+=(d^a^b)+k[15]+530742520|0;c=(c<<16|c>>>16)+d|0;b+=(c^d^a)+k[2]-995338651|0;b=(b<<23|b>>>9)+c|0;a+=(c^(b|~d))+k[0]-198630844|0;a=(a<<6|a>>>26)+b|0;d+=(b^(a|~c))+k[7]+1126891415|0;d=(d<<10|d>>>22)+a|0;c+=(a^(d|~b))+k[14]-1416354905|0;c=(c<<15|c>>>17)+d|0;b+=(d^(c|~a))+k[5]-57434055|0;b=(b<<21|b>>>11)+c|0;a+=(c^(b|~d))+k[12]+1700485571|0;a=(a<<6|a>>>26)+b|0;d+=(b^(a|~c))+k[3]-1894986606|0;d=(d<<10|d>>>22)+a|0;c+=(a^(d|~b))+k[10]-1051523|0;c=(c<<15|c>>>17)+d|0;b+=(d^(c|~a))+k[1]-2054922799|0;b=(b<<21|b>>>11)+c|0;a+=(c^(b|~d))+k[8]+1873313359|0;a=(a<<6|a>>>26)+b|0;d+=(b^(a|~c))+k[15]-30611744|0;d=(d<<10|d>>>22)+a|0;c+=(a^(d|~b))+k[6]-1560198380|0;c=(c<<15|c>>>17)+d|0;b+=(d^(c|~a))+k[13]+1309151649|0;b=(b<<21|b>>>11)+c|0;a+=(c^(b|~d))+k[4]-145523070|0;a=(a<<6|a>>>26)+b|0;d+=(b^(a|~c))+k[11]-1120210379|0;d=(d<<10|d>>>22)+a|0;c+=(a^(d|~b))+k[2]+718787259|0;c=(c<<15|c>>>17)+d|0;b+=(d^(c|~a))+k[9]-343485551|0;b=(b<<21|b>>>11)+c|0;x[0]=a+x[0]|0;x[1]=b+x[1]|0;x[2]=c+x[2]|0;x[3]=d+x[3]|0}function md5blk(s){var md5blks=[],i;for(i=0;i<64;i+=4){md5blks[i>>2]=s.charCodeAt(i)+(s.charCodeAt(i+1)<<8)+(s.charCodeAt(i+2)<<16)+(s.charCodeAt(i+3)<<24)}return md5blks}function md5blk_array(a){var md5blks=[],i;for(i=0;i<64;i+=4){md5blks[i>>2]=a[i]+(a[i+1]<<8)+(a[i+2]<<16)+(a[i+3]<<24)}return md5blks}function md51(s){var n=s.length,state=[1732584193,-271733879,-1732584194,271733878],i,length,tail,tmp,lo,hi;for(i=64;i<=n;i+=64){md5cycle(state,md5blk(s.substring(i-64,i)))}s=s.substring(i-64);length=s.length;tail=[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0];for(i=0;i<length;i+=1){tail[i>>2]|=s.charCodeAt(i)<<(i%4<<3)}tail[i>>2]|=128<<(i%4<<3);if(i>55){md5cycle(state,tail);for(i=0;i<16;i+=1){tail[i]=0}}tmp=n*8;tmp=tmp.toString(16).match(/(.*?)(.{0,8})$/);lo=parseInt(tmp[2],16);hi=parseInt(tmp[1],16)||0;tail[14]=lo;tail[15]=hi;md5cycle(state,tail);return state}function md51_array(a){var n=a.length,state=[1732584193,-271733879,-1732584194,271733878],i,length,tail,tmp,lo,hi;for(i=64;i<=n;i+=64){md5cycle(state,md5blk_array(a.subarray(i-64,i)))}a=i-64<n?a.subarray(i-64):new Uint8Array(0);length=a.length;tail=[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0];for(i=0;i<length;i+=1){tail[i>>2]|=a[i]<<(i%4<<3)}tail[i>>2]|=128<<(i%4<<3);if(i>55){md5cycle(state,tail);for(i=0;i<16;i+=1){tail[i]=0}}tmp=n*8;tmp=tmp.toString(16).match(/(.*?)(.{0,8})$/);lo=parseInt(tmp[2],16);hi=parseInt(tmp[1],16)||0;tail[14]=lo;tail[15]=hi;md5cycle(state,tail);return state}function rhex(n){var s="",j;for(j=0;j<4;j+=1){s+=hex_chr[n>>j*8+4&15]+hex_chr[n>>j*8&15]}return s}function hex(x){var i;for(i=0;i<x.length;i+=1){x[i]=rhex(x[i])}return x.join("")}if(hex(md51("hello"))!=="5d41402abc4b2a76b9719d911017c592"){add32=function(x,y){var lsw=(x&65535)+(y&65535),msw=(x>>16)+(y>>16)+(lsw>>16);return msw<<16|lsw&65535}}if(typeof ArrayBuffer!=="undefined"&&!ArrayBuffer.prototype.slice){(function(){function clamp(val,length){val=val|0||0;if(val<0){return Math.max(val+length,0)}return Math.min(val,length)}ArrayBuffer.prototype.slice=function(from,to){var length=this.byteLength,begin=clamp(from,length),end=length,num,target,targetArray,sourceArray;if(to!==undefined){end=clamp(to,length)}if(begin>end){return new ArrayBuffer(0)}num=end-begin;target=new ArrayBuffer(num);targetArray=new Uint8Array(target);sourceArray=new Uint8Array(this,begin,num);targetArray.set(sourceArray);return target}})()}function toUtf8(str){if(/[\u0080-\uFFFF]/.test(str)){str=unescape(encodeURIComponent(str))}return str}function utf8Str2ArrayBuffer(str,returnUInt8Array){var length=str.length,buff=new ArrayBuffer(length),arr=new Uint8Array(buff),i;for(i=0;i<length;i+=1){arr[i]=str.charCodeAt(i)}return returnUInt8Array?arr:buff}function arrayBuffer2Utf8Str(buff){return String.fromCharCode.apply(null,new Uint8Array(buff))}function concatenateArrayBuffers(first,second,returnUInt8Array){var result=new Uint8Array(first.byteLength+second.byteLength);result.set(new Uint8Array(first));result.set(new Uint8Array(second),first.byteLength);return returnUInt8Array?result:result.buffer}function hexToBinaryString(hex){var bytes=[],length=hex.length,x;for(x=0;x<length-1;x+=2){bytes.push(parseInt(hex.substr(x,2),16))}return String.fromCharCode.apply(String,bytes)}function SparkMD5(){this.reset()}SparkMD5.prototype.append=function(str){this.appendBinary(toUtf8(str));return this};SparkMD5.prototype.appendBinary=function(contents){this._buff+=contents;this._length+=contents.length;var length=this._buff.length,i;for(i=64;i<=length;i+=64){md5cycle(this._hash,md5blk(this._buff.substring(i-64,i)))}this._buff=this._buff.substring(i-64);return this};SparkMD5.prototype.end=function(raw){var buff=this._buff,length=buff.length,i,tail=[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],ret;for(i=0;i<length;i+=1){tail[i>>2]|=buff.charCodeAt(i)<<(i%4<<3)}this._finish(tail,length);ret=hex(this._hash);if(raw){ret=hexToBinaryString(ret)}this.reset();return ret};SparkMD5.prototype.reset=function(){this._buff="";this._length=0;this._hash=[1732584193,-271733879,-1732584194,271733878];return this};SparkMD5.prototype.getState=function(){return{buff:this._buff,length:this._length,hash:this._hash}};SparkMD5.prototype.setState=function(state){this._buff=state.buff;this._length=state.length;this._hash=state.hash;return this};SparkMD5.prototype.destroy=function(){delete this._hash;delete this._buff;delete this._length};SparkMD5.prototype._finish=function(tail,length){var i=length,tmp,lo,hi;tail[i>>2]|=128<<(i%4<<3);if(i>55){md5cycle(this._hash,tail);for(i=0;i<16;i+=1){tail[i]=0}}tmp=this._length*8;tmp=tmp.toString(16).match(/(.*?)(.{0,8})$/);lo=parseInt(tmp[2],16);hi=parseInt(tmp[1],16)||0;tail[14]=lo;tail[15]=hi;md5cycle(this._hash,tail)};SparkMD5.hash=function(str,raw){return SparkMD5.hashBinary(toUtf8(str),raw)};SparkMD5.hashBinary=function(content,raw){var hash=md51(content),ret=hex(hash);return raw?hexToBinaryString(ret):ret};SparkMD5.ArrayBuffer=function(){this.reset()};SparkMD5.ArrayBuffer.prototype.append=function(arr){var buff=concatenateArrayBuffers(this._buff.buffer,arr,true),length=buff.length,i;this._length+=arr.byteLength;for(i=64;i<=length;i+=64){md5cycle(this._hash,md5blk_array(buff.subarray(i-64,i)))}this._buff=i-64<length?new Uint8Array(buff.buffer.slice(i-64)):new Uint8Array(0);return this};SparkMD5.ArrayBuffer.prototype.end=function(raw){var buff=this._buff,length=buff.length,tail=[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],i,ret;for(i=0;i<length;i+=1){tail[i>>2]|=buff[i]<<(i%4<<3)}this._finish(tail,length);ret=hex(this._hash);if(raw){ret=hexToBinaryString(ret)}this.reset();return ret};SparkMD5.ArrayBuffer.prototype.reset=function(){this._buff=new Uint8Array(0);this._length=0;this._hash=[1732584193,-271733879,-1732584194,271733878];return this};SparkMD5.ArrayBuffer.prototype.getState=function(){var state=SparkMD5.prototype.getState.call(this);state.buff=arrayBuffer2Utf8Str(state.buff);return state};SparkMD5.ArrayBuffer.prototype.setState=function(state){state.buff=utf8Str2ArrayBuffer(state.buff,true);return SparkMD5.prototype.setState.call(this,state)};SparkMD5.ArrayBuffer.prototype.destroy=SparkMD5.prototype.destroy;SparkMD5.ArrayBuffer.prototype._finish=SparkMD5.prototype._finish;SparkMD5.ArrayBuffer.hash=function(arr,raw){var hash=md51_array(new Uint8Array(arr)),ret=hex(hash);return raw?hexToBinaryString(ret):ret};return SparkMD5});
`
}

func mooSocket(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, html("localhost", "7122"))
}

func html(serverHost string, serverPort string) string {

	return `<!DOCTYPE HTML>
	<html lang="en">
	<head>
	<meta http-equiv="Content-Type" content="text/html; charset=utf-8">
	<title>Huge File Reader</title>
	</head>
	<body style="margin:0;font-family:sans-serif;background:#4c8;">
		<div id="progress">Drop a file...</div>
		<script>` + jsmd5() + `</script>

<h1>Upload files to ---> ` + computerName + `</h1>
<h3> Server IP ` + WebPage + `</h3>


<p>
<form
	enctype="multipart/form-data"
      action="http://`+getMyLocalIP().String()+`:7122/upl"
      method="post"
	  >
  <label for="filenameformfor">File</label>
  <input type="file" name="filenameformname" id="filenameid" />

  <button type="submit">Submit</button>
</form>
	
	
	
	 </p>

<a href="/admin">Back</a>

		<script>

/*
		function handleSubmit(event) {
  event.preventDefault();

  const data = new FormData(event.target);

  const value = data.get('filenameformname');

  console.log({ value });// probably won't need this
}

var formfilename = handleSubmit()

const form = document.querySelector('form');
form.addEventListener('submit', handleSubmit);
*/	
	(function() {
	var chunkSize = 1024 * 1024 * 4
	var progress = document.getElementById('progress')
	progress.style.background = '#396'
	progress.style.color = '#fff'
	progress.style.fontWeight = 'bold'
	progress.style.padding = '5px'
	progress.style.display = 'block'
	progress.style.height = '20px'
	progress.style.whiteSpace = 'nowrap'
	var socket = new WebSocket('ws://` + serverHost + ":" + serverPort + `/ws')
	socket.onmessage = function(e) {
		if (e.data === 'ready') {
			progress.innerHTML = progress.innerHTML.replace('please wait...', ' upload complete')
			console.log('Received ready event')
		}
		if (e.data === 'exists') {
			console.log('file already exists')
			socket.fileExists = true
		}
	}
	var closeSocket = function() {
		if (socket.readyState !== 1) {
			console.log(socket.readyState)
			socket.close()
			setTimeout(function() {
				socket = new WebSocket('ws://` + serverHost + ":" + serverPort + `/ws')
				if (typeof socket.onclose === 'undefined') socket.onclose = closeSocket
				socket.onmessage = function(e) {
					if (e.data === 'ready') {
						progress.innerHTML = progress.innerHTML.replace('please wait...', ' upload complete')
						console.log('Received ready event')
					}
					if (e.data === 'exists') {
						console.log('file already exists')
						socket.fileExists = true
					}
				}
			}, 5000)
			}
	}
	socket.onclose = closeSocket
	function parseFile(file, options) {
		var fileSize = file.size
		var offset = 0
		var readBlock = null
		var chunkReadCallback = function(data) {
			console.log(data)
			if (!socket.fileExists) socket.send(data)
		}
		var chunkErrorCallback = function(err) {
			console.log('ERROR', err)
		}
		var result = function(msg, count) {
			console.log(msg + ' ' + count)
			socket.fileExists = false
		}
		var onLoadHandler = function(evt) {
			if (evt.target.error == null) {
				offset += evt.loaded
				chunkReadCallback(evt.target.result)
			} else {
				chunkErrorCallback(evt.target.error)
				return
			}
			var percentage = Math.round((offset / fileSize) * 100)
			progress.innerHTML = file.name + ' (MD5=' + socket.uploadChecksum + ') ' +
				' &nbsp; ' + percentage + '% processed, please wait...'
			progress.style.width = percentage + '%'
			if (offset === fileSize) {
				result('Success!', offset)
				socket.send('ready:' +  offset)
				return
			} else if (offset > fileSize) {
				result('Fail!', offset)
				return
			}
			readBlock(offset, chunkSize, file)
		}
		readBlock = function(_offset, length, _file) {
			var r = new FileReader()
			var blob = _file.slice(_offset, length + _offset);
			r.onload = onLoadHandler
			r.readAsArrayBuffer(blob)
		}
		readBlock(offset, chunkSize, file)
	}
	
	function getFileChecksum(file, options) {
		var fileSize = file.size
		var spark = new SparkMD5.ArrayBuffer()
		var offset = 0
		var readBlock = null
		var chunkReadCallback = function(data) {
			spark.append(data)
		}
		var chunkErrorCallback = function(err) {
			console.log('ERROR', err)
		}
		var onLoadHandler = function(evt) {
			if (evt.target.error == null) {
				offset += evt.loaded
				chunkReadCallback(evt.target.result)
			} else {
				chunkErrorCallback(evt.target.error)
				return
			}
			var percentage = Math.round((offset / fileSize) * 100)
			progress.innerHTML = 'Calculating MD5 for ' + file.name + ' &nbsp; ' + percentage + '%'
			progress.style.width = percentage + '%'
			if (offset === fileSize) {
				socket.uploadChecksum = spark.end()
				socket.send('upload:' + socket.uploadChecksum + ':' + file.name)
				parseFile(file)
				return
			} else if (offset > fileSize) {
				return
			}
			readBlock(offset, chunkSize, file)
		}
		readBlock = function(_offset, length, _file) {
			var r = new FileReader()
			var blob = _file.slice(_offset, length + _offset);
			r.onload = onLoadHandler
			r.readAsArrayBuffer(blob)
		}
		readBlock(offset, chunkSize, file)
	}
	
	window.ondragover = function() { return false }
	window.ondrop = function(e) { 
		if (e.dataTransfer.files.length > 0) {
			getFileChecksum(e.dataTransfer.files[0])
		}
		return false 
	}
	})()
		</script>

	</body>
	</html>
	`
}
