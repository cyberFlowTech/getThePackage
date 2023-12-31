package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"

	"strings"

	gojenkins "github.com/jenkins-x/golang-jenkins"
)

const (
	APITOKEN   = "1180585041984f6c3d8c525852041c0772" // uniapp
	JENKINSURL = "https://jenkins.mimo.immo"
)

// 后续有需要替换的场景在这里拓展
func Replace(kind string, value interface{}, filePath string) {
	var command string
	if kind == "appid" {
		appID, _ := value.(string)
		command = fmt.Sprintf("/usr/local/bin/gsed -E -i 's/\"appid\" : \".*\",/\"appid\" : \"%s\",/g' %s", appID, filePath)
	} else if kind == "versionCode" {
		versionCode, _ := value.(int64)
		command = fmt.Sprintf("/usr/local/bin/gsed -E -i 's/\"versionCode\" : .*,/\"versionCode\" : %d,/g' %s", versionCode, filePath)
	} else if kind == "versionName" {
		versionName, _ := value.(string)
		command = fmt.Sprintf("/usr/local/bin/gsed -E -i 's/\"versionName\" : \".*\",/\"versionName\" : \"%s\",/g' %s", versionName, filePath)
	} else if kind == "android_package_name" {
		androidPackageName, _ := value.(string)
		command = fmt.Sprintf("/usr/local/bin/gsed -E -i 's/\"android_package_name\" : \".*?\",/\"android_package_name\" : \"%s\",/g' %s", androidPackageName, filePath)
	} else if kind == "ios_bundle_id" {
		iosBundleId, _ := value.(string)
		command = fmt.Sprintf("/usr/local/bin/gsed -E -i 's/\"ios_bundle_id\" : \"com.mimo.uni.test\",/\"ios_bundle_id\" : \"%s\",/g' %s", iosBundleId, filePath)
	} else if kind == "env" {
		env := value.(string)
		command = fmt.Sprintf("/usr/local/bin/gsed -E -i 's/let PROD = \\w+/let PROD = %s/g' %s", env, filePath)
	} else if kind == "testhost" {
		command = fmt.Sprintf("/usr/local/bin/gsed -E -i 's/http:\\/\\/tt.mimelabs.xyz:8081/https:\\/\\/mimo-dev.mimo.immo/g' %s", filePath)
	} else if kind == "sdk" {
		targetSDKVersion := value.(int64)
		command = fmt.Sprintf("/usr/local/bin/gsed -E -i 's/\"targetSdkVersion\" : .*?/\"targetSdkVersion\" : %d/g' %s", targetSDKVersion, filePath)
	} else {
		fmt.Println("要替换的类型有错误,请检查代码!")
	}
	fmt.Println("command:", command)
	cmd := exec.Command("/bin/bash", "-c", command)
	_, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("/usr/local/bin/gsed config err:", err.Error())
		panic(err)
	}
}

// 动态修改manifest.json和baseurl.json
// go run main.go [prepare] env AndroidPackageName IosBundleID AppID VersionName VersionCode RootPath TargetSDKVersion
func prepare() {

	env := os.Args[2]
	androidPackageName := os.Args[3]
	iosBundleID := os.Args[4]
	appID := os.Args[5]
	versionName := os.Args[6]
	versionCodeStr := os.Args[7]
	rootPath := os.Args[8]
	targetSDKVersionStr := os.Args[9]

	baseUrlPath := fmt.Sprintf("%s/common/const/baseUrl.const.js", rootPath)
	manifestPath := fmt.Sprintf("%s/manifest.json", rootPath)

	// 改为用正则匹配替换

	// 后台API接口
	Replace("env", env, baseUrlPath)
	Replace("testhost", " ", baseUrlPath)

	// APPID
	Replace("appid", appID, manifestPath)
	// 测试维护的版本名称
	Replace("versionName", versionName, manifestPath)
	// 测试维护的版本编码
	versionCode, _ := strconv.ParseInt(versionCodeStr, 10, 64)
	Replace("versionCode", versionCode, manifestPath)
	// 云插件的安卓包名和ios基带名
	Replace("android_package_name", androidPackageName, manifestPath)
	Replace("ios_bundle_id", iosBundleID, manifestPath)
	// sdk
	targetSDKVersion, _ := strconv.ParseInt(targetSDKVersionStr, 10, 64)
	Replace("sdk", targetSDKVersion, manifestPath)

}

// 调用jenkins的api获取日志并下载云打包好的文件
// go run main.go [bothDownload] jobName  androidPackageName  iosPackageName rootPath
func bothDownload() {

	auth := &gojenkins.Auth{
		Username: "linxin",
		ApiToken: APITOKEN,
	}
	jenkins := gojenkins.NewJenkins(auth, JENKINSURL)

	jobName := os.Args[2]
	androidPackageName := os.Args[3]
	iosPackageName := os.Args[4]
	rootPath := os.Args[5]
	isCustom := os.Args[6]
	branch := os.Args[7]

	// 基座包直接移动到文件目录
	if isCustom == "true" {
		cmd := fmt.Sprintf("cp /Users/apple/Documents/git/jenkins/workspace/test_mimo_uniapp_pack/unpackage/debug/android_debug.apk %s/%sDebug.apk", rootPath, branch)
		c := exec.Command("/bin/bash", "-c", cmd)
		err := c.Run()
		if err != nil {
			fmt.Println(err.Error())
			panic(err)
		}
		cmd1 := fmt.Sprintf("cp /Users/apple/Documents/git/jenkins/workspace/test_mimo_uniapp_pack/unpackage/debug/iOS_debug.ipa %s/%sDebug.ipa", rootPath, branch)
		c1 := exec.Command("/bin/bash", "-c", cmd1)
		err1 := c1.Run()
		if err1 != nil {
			fmt.Println(err1.Error())
			panic(err1)
		}

	} else {
		job, err := jenkins.GetJob(jobName)
		if err != nil {
			panic(err)
		}
		job.Url = strings.Replace(job.Url, "http://jenkins:8080", JENKINSURL, 1)

		build, err := jenkins.GetLastBuild(job)
		if err != nil {
			panic(err)
		}
		build.Url = strings.Replace(build.Url, "http://jenkins:8080", JENKINSURL, 1)

		var output []byte
		output, err = jenkins.GetBuildConsoleOutput(build)
		if err != nil {
			panic(err)
		}

		outputStr := string(output)

		reg, _ := regexp.Compile(`.*iOS Appstore 下载地址: https://ide.dcloud.net.cn/build/download/(.*) （注意该地址为临.*`)
		match := reg.FindStringSubmatch(outputStr)
		if len(match) != 2 {
			return
		}
		filePath := fmt.Sprintf("https://ide.dcloud.net.cn/build/download/%s", match[1])
		cmd := fmt.Sprintf("/usr/local/bin/wget -O %s/%s %s", rootPath, iosPackageName, filePath)

		c := exec.Command("/bin/bash", "-c", cmd)
		err = c.Run()
		if err != nil {
			fmt.Println(err.Error())
			panic(err)
		}

		reg1, _ := regexp.Compile(`.*Android自有证书.*? 下载地址: https://ide.dcloud.net.cn/build/download/(.*) （注意该地址为临.*`)
		match1 := reg1.FindStringSubmatch(outputStr)
		if len(match1) != 2 {
			return
		}
		filePath1 := fmt.Sprintf("https://ide.dcloud.net.cn/build/download/%s", match1[1])
		cmd1 := fmt.Sprintf("/usr/local/bin/wget -O %s/%s %s", rootPath, androidPackageName, filePath1)

		c1 := exec.Command("/bin/bash", "-c", cmd1)
		err = c1.Run()
		if err != nil {
			fmt.Println(err.Error())
			panic(err)
		}
	}

}

// go run main.go [prepare/download] jobName packageName

func main() {

	opt := os.Args[1]

	switch opt {
	case "prepare":
		prepare()
	case "bothDownload":
		bothDownload()
	default:
		fmt.Println("请输入正确的命令行参数!prepare:准备 bothDownload:下载")
		return
	}

}
