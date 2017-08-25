package main

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"../myutil"
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

	indexHtml := string(MustAsset("res/index.html"))
	indexHtml = strings.Replace(indexHtml, "<LOGIN/>", loginHtml(w, r), 1)

	html := myutil.MinifyHtml(indexHtml, devMode)

	css, js := myutil.MinifyCssJs(mergeCss(), mergeScripts(), devMode)
	html = strings.Replace(html, "/*.CSS*/", css, 1)
	html = strings.Replace(html, "/*.SCRIPT*/", js, 1)

	w.Write([]byte(html))
}

func loginHtml(w http.ResponseWriter, r *http.Request) string {
	if !writeAuthRequired {
		return ""
	}

	loginCookie := readLoginCookie(r)
	if loginCookie == nil || loginCookie.Name == "" {
		loginCookie, _ = tryLogin(loginCookie, w, r)
	}

	if loginCookie == nil {
		return `<button class="loginButton">Login</button>`
	}

	return `<img class="loginAvatar" src="` + loginCookie.Avatar +
		`"/><span class="loginName">` + loginCookie.Name + `</span>`
}

func tryLogin(loginCookie *CookieValue, w http.ResponseWriter, r *http.Request) (*CookieValue, error) {
	code := r.FormValue("code")
	state := r.FormValue("state")
	log.Println("code:", code, ",state:", state)
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
