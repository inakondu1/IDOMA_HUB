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
	Title      string
	Registered bool
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
			"INSERT INTO users (username, email, password) VALUES ($1, $2, $3)",
			username,
			email,
			string(hashedPassword),
		)
		if err != nil {
			http.Error(w, "Unable to create account. Username or email may already exist.", http.StatusBadRequest)
			log.Println(err)
			return
		}

		http.Redirect(w, r, "/login?registered=1", http.StatusSeeOther)
		return
	}

	tmpl, err := template.ParseFiles("templates/register.html")
	if err != nil {
		http.Error(w, "Unable to load registration page", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	data := PageData{
		Registered: r.URL.Query().Get("registered") == "1",
		Title:      "Create Account - IDOMA HUB",
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Unable to display registration page", http.StatusInternalServerError)
		log.Println(err)
	}
}

func loginPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		usernameOrEmail := r.FormValue("username")
		password := r.FormValue("password")

		var userID int
		var storedPassword string

		err := db.QueryRow(
			"SELECT id, password FROM users WHERE username = $1 OR email = $2",
			usernameOrEmail,
			usernameOrEmail,
		).Scan(&userID, &storedPassword)

		if err != nil {
			http.Error(w, "Invalid username/email or password.", http.StatusUnauthorized)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password))
		if err != nil {
			http.Error(w, "Invalid username/email or password.", http.StatusUnauthorized)
			return
		}

		sessionID, err := createSession(userID)
		if err != nil {
			http.Error(w, "Unable to create login session.", http.StatusInternalServerError)
			log.Println(err)
			return
		}

		setSessionCookie(w, sessionID)
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}

	tmpl, err := template.ParseFiles("templates/login.html")
	if err != nil {
		http.Error(w, "Unable to load login page", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	data := PageData{
		Registered: r.URL.Query().Get("registered") == "1",
		Title:      "Login - IDOMA HUB",
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Unable to display login page", http.StatusInternalServerError)
		log.Println(err)
	}
}

func main() {
	http.Handle("/static/uploads/posts/", http.StripPrefix("/static/uploads/posts/", http.FileServer(http.Dir(postUploadDir()))))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/register", registerPageHandler)
	http.HandleFunc("/login", loginPageHandler)
	http.HandleFunc("/dashboard", dashboardHandler)
	http.HandleFunc("/profile", profileHandler)
	http.HandleFunc("/profile-view", profileViewHandler)
	http.HandleFunc("/profile-picture", profilePictureUploadHandler)
	http.HandleFunc("/friends", friendsHandler)
	http.HandleFunc("/learning", learningHandler)
	http.HandleFunc("/culture", cultureHandler)
	http.HandleFunc("/lessons", lessonsHandler)
	http.HandleFunc("/words", wordsHandler)
	http.HandleFunc("/phrases", phrasesHandler)
	http.HandleFunc("/practice", practiceHandler)
	http.HandleFunc("/names", namesHandler)
	http.HandleFunc("/notifications", notificationsHandler)
	http.HandleFunc("/post", createPostHandler)
	http.HandleFunc("/like", likePostHandler)
	http.HandleFunc("/share", sharePostHandler)
	http.HandleFunc("/comment", createCommentHandler)
	http.HandleFunc("/delete-post", deletePostHandler)
	http.HandleFunc("/logout", logoutHandler)

	fmt.Println("IDOMA HUB is running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
