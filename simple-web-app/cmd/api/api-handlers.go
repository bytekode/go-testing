package main

import (
	"errors"
	"log"
	"net/http"
	"simple-web-app/pkg/data"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

type Credentials struct {
	Username string `json:"email"`
	Password string `json:"password"`
}

func (app *application) authenticate(w http.ResponseWriter, r *http.Request) {

	var cred Credentials

	// read a json payload
	err := app.readJSON(w, r, &cred)
	if err != nil {
		app.errorJSON(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	// look up the user by email address
	user, err := app.DB.GetUserByEmail(cred.Username)
	if err != nil {
		app.errorJSON(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	// check password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(cred.Password))
	if err != nil {
		app.errorJSON(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	// generate tokens
	tokenPairs, err := app.generateTokenPair(user)
	if err != nil {
		app.errorJSON(w, errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "__Host-refresh_token",
		Path:     "/",
		Value:    tokenPairs.RefreshToken,
		Expires:  time.Now().Add(refreshTokenExpiry),
		MaxAge:   int(refreshTokenExpiry.Seconds()),
		SameSite: http.SameSiteStrictMode,
		Domain:   "localhost",
		HttpOnly: true,
		Secure:   true,
	})

	// send token to user

	_ = app.writeJSON(w, http.StatusOK, tokenPairs)

}

func (app *application) refresh(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	refreshToken := r.Form.Get("refresh_token")
	claims := &Claims{}

	_, err = jwt.ParseWithClaims(refreshToken, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(app.JWTSecret), nil
	})

	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	if time.Unix(claims.ExpiresAt.Unix(), 0).Sub(time.Now()) > 30*time.Second {
		app.errorJSON(w, errors.New("refresh token doesnot need renew yet"), http.StatusTooEarly)
		return
	}

	// get user id from claims

	userID, err := strconv.Atoi(claims.Subject)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	user, err := app.DB.GetUser(userID)
	if err != nil {
		app.errorJSON(w, errors.New("unknown user"), http.StatusBadRequest)
		return
	}

	tokenPairs, err := app.generateTokenPair(user)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "__Host-refresh_token",
		Path:     "/",
		Value:    tokenPairs.RefreshToken,
		Expires:  time.Now().Add(refreshTokenExpiry),
		MaxAge:   int(refreshTokenExpiry.Seconds()),
		SameSite: http.SameSiteStrictMode,
		Domain:   "localhost",
		HttpOnly: true,
		Secure:   true,
	})

	_ = app.writeJSON(w, http.StatusOK, tokenPairs)

}

func (app *application) refreshUsingCookie(w http.ResponseWriter, r *http.Request) {
	log.Println("handler refreshUsingCookie executed")
	for _, cookie := range r.Cookies() {
		log.Println("cookie: ", cookie.Name)
		if cookie.Name == "__Host-refresh_token" {
			log.Println("cookie found! ", cookie.Value)
			claims := &Claims{}
			refreshToken := cookie.Value

			_, err := jwt.ParseWithClaims(refreshToken, claims, func(t *jwt.Token) (interface{}, error) {
				return []byte(app.JWTSecret), nil
			})

			if err != nil {
				app.errorJSON(w, err, http.StatusBadRequest)
				return
			}

			// if time.Unix(claims.ExpiresAt.Unix(), 0).Sub(time.Now()) > 30*time.Second {
			// 	app.errorJSON(w, errors.New("refresh token doesnot need renew yet"), http.StatusTooEarly)
			// 	return
			// }

			// get user id from claims

			userID, err := strconv.Atoi(claims.Subject)
			if err != nil {
				app.errorJSON(w, err, http.StatusBadRequest)
				return
			}

			user, err := app.DB.GetUser(userID)
			if err != nil {
				app.errorJSON(w, errors.New("unknown user"), http.StatusBadRequest)
				return
			}

			tokenPairs, err := app.generateTokenPair(user)
			if err != nil {
				app.errorJSON(w, err, http.StatusBadRequest)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:     "__Host-refresh_token",
				Path:     "/",
				Value:    tokenPairs.RefreshToken,
				Expires:  time.Now().Add(refreshTokenExpiry),
				MaxAge:   int(refreshTokenExpiry.Seconds()),
				SameSite: http.SameSiteStrictMode,
				Domain:   "localhost",
				HttpOnly: true,
				Secure:   true,
			})

			_ = app.writeJSON(w, http.StatusOK, tokenPairs)
			return
		}
	}

	app.errorJSON(w, errors.New("unauthorized"), http.StatusUnauthorized)
}

func (app *application) allUsers(w http.ResponseWriter, r *http.Request) {
	users, err := app.DB.AllUsers()
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	app.writeJSON(w, http.StatusOK, users)
}

func (app *application) getUser(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(chi.URLParam(r, "userID"))
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}
	user, err := app.DB.GetUser(userID)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	app.writeJSON(w, http.StatusOK, user)
}

func (app *application) updateUser(w http.ResponseWriter, r *http.Request) {
	var user data.User
	err := app.readJSON(w, r, &user)

	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = app.DB.UpdateUser(user)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (app *application) deleteUser(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(chi.URLParam(r, "userID"))
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = app.DB.DeleteUser(userID)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (app *application) insertUser(w http.ResponseWriter, r *http.Request) {
	var user data.User
	err := app.readJSON(w, r, &user)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}
	_, err = app.DB.InsertUser(user)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
