package main

import "net/http"

func authOk(r *http.Request) bool {
	if !writeAuthRequired {
		return true
	}

	loginCookie := readLoginCookie(r)
	if loginCookie != nil && loginCookie.Name != "" {
		return true
	}

	return false
}
