// main.go
package main

import (
	"bytes"
	"fmt"
	"github.com/Unknwon/goconfig"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
)

var debug = false
var checkUrl = "http://180.97.33.107"
//var wlanName = "nfsysugz"
var wlanName = "nfsysugz2"
var loginUrl = "http://219.136.125.139/portalAuthAction.do"
var username = ""
var password = ""

type loginInfo struct {
	wlanuserip string
	wlancname  string
	auth_type  string
	wlanacIp   string
}

func getDefLoginInfo() *loginInfo {
	return &loginInfo{auth_type: "PAP"}
}

func initSetting() bool {
	cfg, err := goconfig.LoadConfigFile("config.ini")
	checkErr(err)
	username, err = cfg.GetValue("auth", "username")
	checkErr(err)
	password, err = cfg.GetValue("auth", "password")
	if username != "" && password != "" {
		return true
	}
	return false
}

func fetch(rurl string) (string, *url.URL) {
	return fetchWithRef(rurl, "")
}

func fetchWithRef(rurl string, referer string) (string, *url.URL) {
	client := &http.Client{}

	request, error := http.NewRequest("GET", rurl, nil)
	checkErr(error)

	request.Header.Set("User-Agent", "Supplicant")
	request.Header.Set("Accpet", "image/jpeg, application/x-ms-application, image/gif, application/xaml+xml, image/pjpeg, application/x-ms-xbap, application/vnd.ms-excel, application/vnd.ms-powerpoint, application/msword, */*")
	request.Header.Set("Accept-Language", "zh-CN,zh;q=0.8")
	request.Header.Set("Accept-Encoding", "deflate")
	request.Header.Set("Content-Type", "application/x-w-form-urlencoded")
	if referer != "" {
		request.Header.Set("Referer", referer)
	}

	response, error := client.Do(request)
	checkErr(error)
	//如果处理失败则回调自身再试一次
	if error != nil {
		return fetch(rurl)
	}

	defer response.Body.Close()
	content, error := ioutil.ReadAll(response.Body)
	realUrl := response.Request.URL

	checkErr(error)

	return string(content), realUrl

}

func fetchPostWithRef(rurl string, info loginInfo, referer string) string {
	client := &http.Client{}

	data := url.Values{}
	data.Set("userid", username)
	data.Set("passwd", password)
	data.Set("wlanuserip", info.wlanuserip)
	data.Set("auth_type", info.auth_type)
	data.Set("wlanacname", info.wlancname)
	data.Set("wlanacIp", info.wlanacIp)

	request, error := http.NewRequest("POST", rurl, bytes.NewBufferString(data.Encode()))
	if error != nil {
		fmt.Printf("error [%s]", error)
		return ""
	}

	request.Header.Set("User-Agent", "Supplicant")
	request.Header.Set("Accpet", "image/jpeg, application/x-ms-application, image/gif, application/xaml+xml, image/pjpeg, application/x-ms-xbap, application/vnd.ms-excel, application/vnd.ms-powerpoint, application/msword, */*")
	request.Header.Set("Accept-Encoding", "deflate")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept-Language", "zh-CN,zh;q=0.8")
	if referer != "" {
		request.Header.Set("Referer", referer)
	}

	response, error := client.Do(request)
	checkErr(error)
	if error != nil {
		return fetchPostWithRef(rurl, info, referer)
	}

	defer response.Body.Close()
	content, error := ioutil.ReadAll(response.Body)
	checkErr(error)

	//fmt.Printf("received from[%s], length[%d]\n", url, len(content))
	return string(content)
}

func main() {
	defer cleanAndExit()
	fmt.Println("Welcome to NFU third-party client")
	fmt.Println("This client is for the [young network client]")
	fmt.Println("Init.....")
	initSucc := initSetting()
	if initSucc {
		fmt.Printf("init success [%s]\n", username)
	} else {
		fmt.Println("init failed!")
		return
	}

	fmt.Println("Checking.....")
	logined, realUrl := isLogin()
	if logined {
		fmt.Println("You have already logined!")
		return
	}
	fmt.Println("Try to login...")
	loginSucc, msg := login(realUrl)
	if !loginSucc {
		fmt.Printf("Login failed [%s]\n", msg)
		return
	}
	fmt.Println("Login Success")
}

func cleanAndExit() {
	fmt.Println("Please press any key to exit.")
	b := make([]byte, 1)
	os.Stdin.Read(b)
}

func isLogin() (bool, *url.URL) {
	_, realUrl := fetch(checkUrl)
	//not redirect if logined
	if realUrl.String() == checkUrl {
		return true, nil
	}
	return false, realUrl
}

func login(realUrl *url.URL) (bool, string) {
	values := realUrl.Query()
	wlanuserip := values["wlanuserip"][0]
	wlancname := values["wlanacname"][0]
	wlanacIp := values["wlanacip"][0]

	if wlancname != wlanName {
		return false, "You are not using NFU network!"
	}
	fmt.Printf("Your ip is [%s], Logining...\n", wlanuserip)

	info := getDefLoginInfo()
	info.wlanacIp = wlanacIp
	info.wlancname = wlancname
	info.wlanuserip = wlanuserip

	content := fetchPostWithRef(loginUrl, *info, realUrl.String())
	contentArr, _ := GbkToUtf8([]byte(content))
	content = string(contentArr)
	if debug {
		fmt.Println(content)
	}
	webErr := getWebError(content)
	if webErr != "" {
		return false, webErr
	}
	//todo 判断返回结果
	return true, content
}

func getWebError(content string) string {
	//alert('认证超时，请稍后再重认证！')
	reg := regexp.MustCompile(`alert\(\'([^<].+?)\'\)`)
	regArray := reg.FindStringSubmatch(content)
	if len(regArray) < 2 {
		return ""
	}
	return regArray[1]
}

func checkErr(err error) {
	if err != nil {
		fmt.Printf("error [%s]\n", err)
	}
}

func GbkToUtf8(s []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewDecoder())
	d, e := ioutil.ReadAll(reader)
	if e != nil {
		return nil, e
	}
	return d, nil
}

func Utf8ToGbk(s []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewEncoder())
	d, e := ioutil.ReadAll(reader)
	if e != nil {
		return nil, e
	}
	return d, nil
}
