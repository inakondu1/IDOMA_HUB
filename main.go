package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
)

var db = initDatabase()

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

func registerPageHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/register.html")
	if err != nil {
		http.Error(w, "Unable to load registration page", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	data := PageData{
		Title: "Create Account - IDOMA HUB",
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Unable to display registration page", http.StatusInternalServerError)
		log.Println(err)
	}
}

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/register", registerPageHandler)

	fmt.Println("IDOMA HUB is running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
