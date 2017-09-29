package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
)

func main() {
	http.HandleFunc(g_config.ContextPath+"/", goIpAllowIndexHandler)
	http.HandleFunc(g_config.ContextPath+"/ipAllow", goIpAllowHandler) // 设置IP权限

	sport := strconv.Itoa(g_config.ListenPort)
	fmt.Println("start to listen at ", sport)

	http.ListenAndServe(":"+sport, nil)
}

func goIpAllowHandler(w http.ResponseWriter, req *http.Request) {
	officeIp := strings.TrimSpace(req.FormValue("officeIp"))
	envs := strings.TrimSpace(req.FormValue("envs"))
	fmt.Println("officeIp:", officeIp, ",env:", envs)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	csrfToken := RandomString(10)

	writeCsrfTokenCookie(w, csrfToken, officeIp, envs)
	url := createWxQyLoginUrl(g_config.RedirectUri, csrfToken)
	log.Println("wx login url:", url)

	w.Write([]byte(url))
}

func ipAllow(r *http.Request, cookie *CookieValue) string {
	_, allowedIps := isIpAlreadyAllowed(cookie.Envs, cookie.OfficeIp)

	if allowedIps == "" {
		allowedIps = cookie.OfficeIp
	} else {
		allowedIps += "," + cookie.OfficeIp
	}

	out, err := exec.Command("/bin/bash", g_config.UpdateFirewallShell, cookie.Envs, allowedIps).Output()
	if err != nil {
		return `设置失败，执行SHELL错误` + err.Error()
	}

	shellOut := string(out)
	fmt.Println(shellOut)
	writeAllowIpFile(cookie.Envs, allowedIps)

	return `设置成功`
}

func isIpAlreadyAllowed(envs, ip string) (bool, string) {
	content, err := ioutil.ReadFile(envs + "-AllowIps.txt")
	if err != nil {
		return false, ""
	}

	strContent := string(content)
	alreadyAllowed := strings.Contains(strContent, ip)
	return alreadyAllowed, strContent
}

func writeAllowIpFile(env, content string) {
	ioutil.WriteFile(env+"-AllowIps.txt", []byte(content), 0644)
}

func goIpAllowIndexHandler(w http.ResponseWriter, r *http.Request) {
	ok, cookie := login(w, r)
	msg := ""
	if ok {
		msg = ipAllow(r, cookie)
	}
	clearCookie(w)

	envCheckboxes := ""
	for _, env := range g_config.Envs {
		envCheckboxes += fmt.Sprintf("<input class='env' type='checkbox' checked value='%v'>%v</input><br/>", env, env)
	}
	resp := `
<html>
<body>
<div style="text-align: center;white-space:nowrap;">
<br/>
请关闭所有代理，防止IP识别错误。<br/>请检查以下显示的IP是：
<span id='myip'></span>
<br/>
<iframe id="iframe" src="http://2017.ip138.com/ic.asp" rel="nofollow" frameborder="0" scrolling="no"
 style="width:100%;height:30px"></iframe>
<br/><div style="width: 200px;margin: 0 auto;text-align: left;">` + envCheckboxes + `</div>
<br/><br/>
<button onclick="setIpAllow()" style="font-size: 14px; padding: 3px 106px; ">设置</button>
</div>

</body>
<script>
function setIpAllow() {
	minAjax({
		url:"/ipAllow",
		type:"POST",
		data: {
			envs: getCheckedValues('env'),
			officeIp:$('myip').innerText
		},
		success: function(redirectUrl) {
			window.location = redirectUrl
		}
	})
}

function getCheckedValues(checkboxClass) {
	var checkedValue = []
	var inputElements = document.getElementsByClassName(checkboxClass)
	for(var i = 0; inputElements[i]; ++i){
		if(inputElements[i].checked){
			checkedValue.push(inputElements[i].value)
		}
	}
	return checkedValue.join(',')
}

function $(id) {
	return document.getElementById(id)
}
`

	/*|--(A Minimalistic Pure JavaScript Header for Ajax POST/GET Request )--|
	  |--Author : flouthoc (gunnerar7@gmail.com)(http://github.com/flouthoc)--|
	*/

	resp += `
function initXMLhttp() {
		if (window.XMLHttpRequest) { // code for IE7,firefox chrome and above
			return new XMLHttpRequest()
		} else { // code for Internet Explorer
			return new ActiveXObject("Microsoft.XMLHTTP")
		}
	}

function minAjax(config) {
`
	/*
		Config Structure
		url:"reqesting URL"
		type:"GET or POST"
		async: "(OPTIONAL) True for async and False for Non-async | By default its Async"
		data: "(OPTIONAL) another Nested Object which should contains reqested Properties in form of Object Properties"
		success: "(OPTIONAL) Callback function to process after response | function(data,status)"
	*/

	resp += `
config.async = config.async || true

	var xmlhttp = initXMLhttp()
	xmlhttp.onreadystatechange = function() {
		if (xmlhttp.readyState == 4 && xmlhttp.status == 200) {
			config.success(xmlhttp.responseText, xmlhttp.readyState)
		}
	}

	var sendString = [], sendData = config.data
	if ( typeof sendData === "string" ){
		var tmpArr = String.prototype.split.call(sendData,'&')
		for(var i = 0, j = tmpArr.length; i < j; i++){
			var datum = tmpArr[i].split('=')
			sendString.push(encodeURIComponent(datum[0]) + "=" + encodeURIComponent(datum[1]))
		}
	} else if( typeof sendData === 'object' && !( sendData instanceof String || (FormData && sendData instanceof FormData) ) ){
		for (var k in sendData) {
			var datum = sendData[k]
			if ( Object.prototype.toString.call(datum) == "[object Array]" ){
				for(var i = 0, j = datum.length; i < j; i++) {
					sendString.push(encodeURIComponent(k) + "[]=" + encodeURIComponent(datum[i]))
				}
			} else {
				sendString.push(encodeURIComponent(k) + "=" + encodeURIComponent(datum))
			}
		}
	}
	sendString = sendString.join('&')

	if (config.type == "GET") {
		xmlhttp.open("GET", config.url + "?" + sendString, config.async)
		xmlhttp.send()
	} else if (config.type == "POST") {
		xmlhttp.open("POST", config.url, config.async)
		xmlhttp.setRequestHeader("Content-type", "application/x-www-form-urlencoded")
		xmlhttp.send(sendString)
	}
}

minAjax({
		url:"http://icanhazip.com",
		type:"GET",
		data:{},
		success: function(data){
			$('myip').innerText = data.trim()
		}
	})
`
	if ok {
		resp += `
alert('` + msg + `')
`
	}

	resp += `
</script>
</html>
`

	w.Write([]byte(resp))
}
