package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/astaxie/beego/toolbox"
	"golang.org/x/net/html/charset"
)

/*
<?xml version="1.0" encoding="gb2312"?>
<property>
<returncode>200</returncode>
<key>intelleast.com</key>
<original>211 : Domain name is not available</original>
</property>

*/

var (
	domains    *string
	dintalkURL *string
	taskSpec   *string
)

type SConfig struct {
	Property   xml.Name `xml:"property"`   // 指定最外层的标签为property
	Returncode int      `xml:"returncode"` // 读取returncode
	Key        string   `xml:"key"`
	Original   string   `xml:"original"`
}

func main() {
	domains = flag.String("d", "", "要监视的域名列表，域名之间用英文逗号分隔。")
	dintalkURL = flag.String("w", "", "要提醒的钉钉机器人的webhook地址，多个地址之间用逗号分隔")
	taskSpec = flag.String("t", "0/10 * * * * *", "定时任务的定时间隔字符，具体参考：https://beego.me/docs/module/toolbox.md#task")
	flag.Parse()

	log.Println("begin task ")
	//创建定时任务
	////  0/30 * * * * *                        每 30 秒 执行
	////  0 0 * * * *　　　　　　　　               毎时 0 分 每隔 1 小时 执行
	dingdingNewBot := toolbox.NewTask("dingding-news-bot", *taskSpec, queryDomains)
	//添加定时任务
	toolbox.AddTask("dingding-news-bot", dingdingNewBot)
	//启动定时任务
	toolbox.StartTask()
	defer toolbox.StopTask()
	select {}
}

//查询单个域名的注册状态
func queryStatues(domain string) error {
	resp, err := http.Get("http://panda.www.net.cn/cgi-bin/check.cgi?area_domain=" + domain)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	//fmt.Println(string(body))
	v := SConfig{}
	decoder := xml.NewDecoder(bytes.NewReader(body))
	decoder.CharsetReader = func(c string, i io.Reader) (io.Reader, error) {
		return charset.NewReaderLabel(c, i)
	}
	decoder.Decode(&v)
	if v.Returncode != 200 {
		return errors.New("whois query faile ")
	}
	s := strings.Split(v.Original, ":")
	if strings.Replace(s[0], " ", "", -1) == "210" {
		dingTalkSellWaring(domain)
	}
	return nil
}

//向钉钉发送告警信息
func dingTalkSellWaring(domain string) {
	dintalkURLs := strings.Split(*dintalkURL, ",")
	for _, w := range dintalkURLs {
		u := strings.Replace(w, " ", "", -1)
		sendMsgToDingtalk(domain+" 可以注册了！", u)
	}
}

//查询多个域名状态
func queryDomains() error {
	log.Println("run task ")
	subsCodes := strings.Split(*domains, ",")
	for _, s := range subsCodes {
		d := strings.Replace(s, " ", "", -1)
		queryStatues(d)
	}
	return nil
}

//发送信息到钉钉
func sendMsgToDingtalk(content string, webHookURL string) error {
	if content != "" {
		formt := `
        {
            "msgtype": "text",
             "text": {
                "content": "%s"
            }
        }`
		body := fmt.Sprintf(formt, content)
		jsonValue := []byte(body)
		resp, err := http.Post(webHookURL, "application/json", bytes.NewBuffer(jsonValue))
		if err != nil {
			return err
		}
		log.Println(resp)
	}
	return nil
}
