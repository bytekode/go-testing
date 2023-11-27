package main

import (
	"errors"
	"fmt"
	"net/http"
	"simple-web-app/pkg/data"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const jwtTokenExpiry = time.Minute * 15
const refreshTokenExpiry = time.Hour * 24

type TokenPairs struct {
	Token        string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Claims struct {
	UserName string `json:"name"`
	jwt.RegisteredClaims
}

func (app *application) getTokenFromHeaderAndVerify(w http.ResponseWriter, r *http.Request) (string, *Claims, error) {
	// add a header
	w.Header().Add("Vary", "Authorization")

	// get the authorization header
	authHeader := r.Header.Get("Authorization")

	// sanity check
	if authHeader == "" {
		return "", nil, errors.New("no auth header")
	}

	// split the header on spaces
	headerParts := strings.Split(authHeader, " ")
	if len(headerParts) != 2 {
		return "", nil, errors.New("invalid auth header")
	}

	// check to see if we have word "Bearer"
	if headerParts[0] != "Bearer" {
		return "", nil, errors.New("unauthorized: no Bearer")
	}

	token := headerParts[1]

	// declare an empty claims var
	claims := &Claims{}

	// parse the token
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		// validate the signing algorithm

		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		return []byte(app.JWTSecret), nil
	})

	// check for an error, note that this catches expired tokens as well.

	if err != nil {
		if strings.HasPrefix(err.Error(), "token is expired by") {
			return "", nil, errors.New("expired token")
		}
		return "", nil, err
	}

	// make sure that we issued the token
	if claims.Issuer != app.Domain {
		return "", nil, errors.New("incorrect issuer")
	}

	return token, claims, nil
}

func (app *application) generateTokenPair(user *data.User) (TokenPairs, error) {
	// Create the token
	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)
	claims["name"] = fmt.Sprintf("%s %s", user.FirstName, user.LastName)
	claims["sub"] = fmt.Sprint(user.ID)
	claims["aud"] = app.Domain
	claims["iss"] = app.Domain

	if user.IsAdmin == 1 {
		claims["admin"] = true
	} else {
		claims["admin"] = false
	}

	// set the expiry

	claims["exp"] = time.Now().Add(jwtTokenExpiry).Unix()

	// create the singed token

	signedAccessToken, err := token.SignedString([]byte(app.JWTSecret))
	if err != nil {
		return TokenPairs{}, err
	}

	// create the refresh token
	refreshToken := jwt.New(jwt.SigningMethodHS256)
	refreshTokenClaims := refreshToken.Claims.(jwt.MapClaims)
	refreshTokenClaims["sub"] = fmt.Sprint(user.ID)
	refreshTokenClaims["exp"] = time.Now().Add(refreshTokenExpiry).Unix()

	signedRefreshToken, err := refreshToken.SignedString([]byte(app.JWTSecret))
	if err != nil {
		return TokenPairs{}, err
	}

	return TokenPairs{
		Token:        signedAccessToken,
		RefreshToken: signedRefreshToken,
	}, nil
}
