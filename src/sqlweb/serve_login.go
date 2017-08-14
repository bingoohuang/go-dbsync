package main

import "net/http"

func serveLogin(w http.ResponseWriter, req *http.Request) {
	csrfToken := RandomString(10)
	writeCsrfTokenCookie(w, csrfToken)
	url := createWxQyLoginUrl(cropId, agentId, redirectUri, csrfToken)
	redirectWxQyLogin(w, req, url)
}
