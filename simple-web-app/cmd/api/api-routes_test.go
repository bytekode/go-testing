package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func Test_app_routes(t *testing.T) {
	var registered = []struct {
		route  string
		method string
	}{
		{route: "/auth", method: "POST"},
		{route: "/refresh-token", method: "POST"},
		{route: "/users/", method: "GET"},
		{route: "/users/{userID}", method: "GET"},
		{route: "/users/{userID}", method: "DELETE"},
		{route: "/users/", method: "PATCH"},
		{route: "/users/", method: "PUT"},
	}
	mux := app.routes()

	chiRoutes := mux.(chi.Routes)
	for _, route := range registered {
		if !routeExists(route.route, route.method, chiRoutes) {
			t.Errorf("route %s is not registered", route.route)
		}
	}
}

func routeExists(testRoute, testMethod string, chiRoutes chi.Routes) bool {
	found := false

	_ = chi.Walk(chiRoutes, func(method, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		if strings.EqualFold(method, testMethod) && strings.EqualFold(route, testRoute) {
			found = true
		}
		return nil
	})
	return found
}
