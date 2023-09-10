package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	fmt.Println("lets muck arround with the fs library")



err := os.Chdir("..")
if err != nil {
	log.Println(err)
}

err = os.Chdir("..")
if err != nil {
	log.Println(err)
}


wd, err := os.Getwd()

if err != nil {
	log.Println(err)
}
	fsys := os.DirFS(wd)

fmt.Println(fsys)

}
