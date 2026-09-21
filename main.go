package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"golang.org/x/crypto/bcrypt"
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
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		email := r.FormValue("email")
		password := r.FormValue("password")
		confirmPassword := r.FormValue("confirm_password")

		if password != confirmPassword {
			http.Error(w, "Passwords do not match.", http.StatusBadRequest)
			return
		}

		if len(password) < 8 {
			http.Error(w, "Password must be at least 8 characters.", http.StatusBadRequest)
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Unable to secure password.", http.StatusInternalServerError)
			log.Println(err)
			return
		}

		_, err = db.Exec(
			"INSERT INTO users (username, email, password) VALUES (?, ?, ?)",
			username,
			email,
			string(hashedPassword),
		)
		if err != nil {
			http.Error(w, "Unable to create account. Username or email may already exist.", http.StatusBadRequest)
			log.Println(err)
			return
		}

		fmt.Fprintln(w, "Account created successfully!")
		return
	}

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
