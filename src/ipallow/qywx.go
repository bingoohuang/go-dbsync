package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type ExpiredStore struct {
	Value   string
	Expired string
}

var (
	accessToken            string
	accessTokenExpiredTime time.Time
	accessTokenMutex       sync.Mutex
)

type TokenResult struct {
	ErrCode          int    `json:"errcode"`
	ErrMsg           string `json:"errmsg"`
	AccessToken      string `json:"access_token"`
	ExpiresInSeconds int    `json:"expires_in"`
}

func getAccessToken(corpId, corpSecret string) (string, error) {
	accessTokenMutex.Lock()
	defer accessTokenMutex.Unlock()
	if accessToken != "" && accessTokenExpiredTime.After(time.Now()) {
		return accessToken, nil
	}

	url := "https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=" + corpId + "&corpsecret=" + corpSecret
	log.Println("url:", url)
	resp, err := http.Get(url)
	log.Println("resp:", resp, ",err:", err)
	if err != nil {
		accessToken = ""
		return "", err
	}

	body := readObjectBytes(resp.Body)

	var tokenResult TokenResult
	json.Unmarshal(body, &tokenResult)
	if tokenResult.ErrCode != 0 {
		return "", errors.New(tokenResult.ErrMsg)
	}

	accessToken = tokenResult.AccessToken
	accessTokenExpiredTime = time.Now().Add(time.Duration(tokenResult.ExpiresInSeconds) * time.Second)

	return accessToken, nil
}

func createWxQyLoginUrl(redirectUri, csrfToken string) string {
	return "https://open.work.weixin.qq.com/wwopen/sso/qrConnect?appid=" +
		g_config.WxCorpId + "&agentid=" + strconv.FormatInt(g_config.WxAgentId, 10) + "&redirect_uri=" + redirectUri + "&state=" + csrfToken
}

type CookieValue struct {
	OfficeIp    string
	Envs        string
	CsrfToken   string
	ExpiredTime string
}

func writeCsrfTokenCookie(w http.ResponseWriter, csrfToken, officeIp, envs string) {
	cookieVal, err := json.Marshal(CookieValue{
		OfficeIp:    officeIp,
		Envs:        envs,
		CsrfToken:   csrfToken,
		ExpiredTime: time.Now().Add(time.Duration(24) * time.Hour).Format(time.RFC3339),
	})
	if err != nil {
		log.Println("json cookie error", err)
	}

	json := string(cookieVal)
	log.Println("csrf json:", json)
	cipher, err := CBCEncrypt(g_config.EncryptKey, json)
	if err != nil {
		log.Println("CBCEncrypt cookie error", err)
	}

	cookie := http.Cookie{Name: g_config.CookieName, Value: cipher, Path: "/", MaxAge: 86400}
	http.SetCookie(w, &cookie)
}

func clearCookie(w http.ResponseWriter) {
	cookie := http.Cookie{Name: g_config.CookieName, Value: "", Path: "/", Expires: time.Now().AddDate(-1, 0, 0)}
	http.SetCookie(w, &cookie)
}

func readLoginCookie(r *http.Request) *CookieValue {
	cookie, _ := r.Cookie(g_config.CookieName)
	if cookie == nil {
		return nil
	}

	log.Println("cookie value:", cookie.Value)
	decrypted, _ := CBCDecrypt(g_config.EncryptKey, cookie.Value)
	if decrypted == "" {
		return nil
	}

	var cookieValue CookieValue
	err := json.Unmarshal([]byte(decrypted), &cookieValue)
	if err != nil {
		log.Println("unamrshal error:", err)
		return nil
	}

	log.Println("cookie parsed:", cookieValue, ",ExpiredTime:", cookieValue.ExpiredTime)

	expired, err := time.Parse(time.RFC3339, cookieValue.ExpiredTime)
	if err != nil {
		log.Println("time.Parse:", err)
	}
	if err != nil || expired.Before(time.Now()) {
		return nil
	}

	return &cookieValue
}

type WxLoginUserId struct {
	UserId  string `json:"UserId"`
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
}

func getLoginUserId(accessToken, code string) (string, error) {
	url := "https://qyapi.weixin.qq.com/cgi-bin/user/getuserinfo?access_token=" + accessToken + "&code=" + code
	log.Println("url:", url)
	resp, err := http.Get(url)
	log.Println("resp:", resp, ",err:", err)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var wxLoginUserId WxLoginUserId
	err = json.Unmarshal(body, &wxLoginUserId)
	if err != nil {
		return "", err
	}
	if wxLoginUserId.UserId == "" {
		return "", errors.New(string(body))
	}

	return wxLoginUserId.UserId, nil
}

type WxUserInfo struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
	UserId string `json:"userid"`
}

func getUserInfo(accessToken, userId string) (*WxUserInfo, error) {
	url := "https://qyapi.weixin.qq.com/cgi-bin/user/get?access_token=" + accessToken + "&userid=" + userId
	log.Println("url:", url)
	resp, err := http.Get(url)
	log.Println("resp:", resp, ",err:", err)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var wxUserInfo WxUserInfo
	err = json.Unmarshal(body, &wxUserInfo)
	if err != nil {
		return nil, err
	}

	return &wxUserInfo, nil
}

func login(w http.ResponseWriter, r *http.Request) (bool, *CookieValue) {
	loginCookie := readLoginCookie(r)
	if loginCookie == nil {
		return false, nil
	}

	ok, _ := tryLogin(loginCookie, w, r)
	return ok, loginCookie
}

func tryLogin(loginCookie *CookieValue, w http.ResponseWriter, r *http.Request) (bool, error) {
	code := r.FormValue("code")
	state := r.FormValue("state")
	log.Println("code:", code, ",state:", state)
	if loginCookie != nil && code != "" && state == loginCookie.CsrfToken {
		accessToken, err := getAccessToken(g_config.WxCorpId, g_config.WxCorpSecret)
		if err != nil {
			return false, err
		}
		userId, err := getLoginUserId(accessToken, code)
		if err != nil {
			return false, err
		}
		_, err = getUserInfo(accessToken, userId)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	return false, errors.New("no login")
}
