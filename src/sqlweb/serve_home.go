package main

import (
	"errors"
	"log"
	"net/http"
	"strings"
)

func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != contextPath+"/" {
		http.Error(w, "Not found", 404)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	index := string(MustAsset("res/index.html"))
	index = strings.Replace(index, "/*.CSS*/", mergeCss(), 1)
	index = strings.Replace(index, "/*.SCRIPT*/", mergeScripts(), 1)

	index = strings.Replace(index, "<LOGIN/>", loginHtml(w, r), 1)

	w.Write([]byte(index))
}

func loginHtml(w http.ResponseWriter, r *http.Request) string {
	loginCookie := readLoginCookie(r)
	if loginCookie != nil && loginCookie.Name != "" && loginCookie.Avatar != "" {
		return `<img src="` + loginCookie.Avatar + `"/><div class="loginName">` + loginCookie.Name + `</div>`
	}

	loginCookie, err := tryLogin(loginCookie, w, r)
	if err != nil {
		log.Println("login error:", err)
		return `<button class="loginButton">Login</button>`
	}

	if loginCookie != nil {
		return `<img src="` + loginCookie.Avatar + `"/><div class="loginName">` + loginCookie.Name + `</div>`
	}

	return `<button class="loginButton">Login</button>`
}

func tryLogin(loginCookie *CookieValue, w http.ResponseWriter, r *http.Request) (*CookieValue, error) {
	code := r.FormValue("code")
	state := r.FormValue("state")
	if loginCookie != nil && code != "" && state == loginCookie.CsrfToken {
		accessToken, err := getAccessToken(corpId, corpSecret)
		if err != nil {
			return nil, err
		}
		userId, err := getLoginUserId(accessToken, code)
		if err != nil {
			return nil, err
		}
		userInfo, err := getUserInfo(accessToken, userId)
		if err != nil {
			return nil, err
		}
		cookie := writeUserInfoCookie(w, userInfo)
		return cookie, nil
	}

	return nil, errors.New("no login")
}
