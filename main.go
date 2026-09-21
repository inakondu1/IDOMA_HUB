package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
)

type PageData struct {
	Title string
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/home.html")
	if err != nil {
		http.Error(w, "Unable to load page", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	data := PageData{
		Title: "IDOMA HUB",
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Unable to display page", http.StatusInternalServerError)
		log.Println(err)
	}
}

func main() {
	http.HandleFunc("/", homeHandler)

	fmt.Println("IDOMA HUB is running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
