package main

import (
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"regexp"
	"strconv"
	"strings"

	"github.com/robfig/cron"
	gcfg "gopkg.in/gcfg.v1"
)

type Monitor struct {
	Title   string
	Url     string
	Pattern string
}

func (m *Monitor) Print() {
	fmt.Println("Title :", m.Title)
	fmt.Println("Url :", m.Url)
	fmt.Println("Pattern :", m.Pattern)
}

type Mail struct {
	Host     string
	User     string
	Password string
	To       string
}

func (m *Mail) Print() {
	fmt.Println("Host :", m.Host)
	fmt.Println("User :", m.User)
	fmt.Println("Password :", m.Password)
	fmt.Println("To :", m.To)
}

type MonitorConfig struct {
	Monitor Monitor
	Mail    Mail
}

func (m *MonitorConfig) Print() {
	m.Monitor.Print()
	m.Mail.Print()
}

var g_config MonitorConfig

func main() {
	//var config MonitorConfig
	err := gcfg.ReadFileInto(&g_config, "monitor.ini")
	if err != nil {
		fmt.Printf("Failed to parse config file: %v", err)
		return
	}

	fmt.Println("-------------------------")
	g_config.Print()
	fmt.Println("-------------------------")
	//return

	// 这个监控精确到秒
	// 如果想用更精确的就需要使用golang本身的定时器
	c := cron.New()
	//spec := "@hourly"
	spec := "*/5 * * * * ?"
	c.AddFunc(spec, MonitorUrl)
	c.Start()
	defer c.Stop()

	select {}
}

func MonitorUrl() {
	url := g_config.Monitor.Url
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
		log.Fatal(err)
	}
	if resp.StatusCode == http.StatusOK {
		fmt.Println(resp.StatusCode)
	}
	defer resp.Body.Close()

	buf := make([]byte, 1024)
	r := regexp.MustCompile(g_config.Monitor.Pattern)
	rr := regexp.MustCompile("\\d+(.\\d{2})?")

	for {
		n, _ := resp.Body.Read(buf)
		if 0 == n {
			break
		}

		txt := string(buf[:n])
		result := r.FindString(txt)
		if result != "" {
			fmt.Printf("result :[%s]\n", result)
			rr.ReplaceAllFunc([]byte(result), MonitorPrice)
		}

	}
}

var cur_price float64 = 0

func MonitorPrice(result []byte) []byte {
	price, err := strconv.ParseFloat(string(result), 32)
	if err != nil {
		fmt.Printf("MonitorPrice error!:%v", err)
		return result
	}

	if cur_price == price {
		fmt.Println("MonitorPrice price not changed!")
		return result
	}

	host := g_config.Mail.Host
	user := g_config.Mail.User
	password := g_config.Mail.Password
	to := g_config.Mail.To
	subject := g_config.Monitor.Title + " [最新价格:" + string(result) + "]"
	err = SendToMail(user, password, host, to, subject, g_config.Monitor.Url, "html")
	if err != nil {
		fmt.Println("Send mail error!")
		fmt.Println(err)
	} else {
		fmt.Println("Send mail success!")
		cur_price = price
	}
	return result
}

func SendToMail(user, password, host, to, subject, body, mailtype string) error {
	hp := strings.Split(host, ":")
	auth := smtp.PlainAuth("", user, password, hp[0])
	var content_type string
	if mailtype == "html" {
		content_type = "Content-Type: text/" + mailtype + "; charset=UTF-8"
	} else {
		content_type = "Content-Type: text/plain" + "; charset=UTF-8"
	}

	msg := []byte("To: " + to + "\r\nFrom: " + user + ">\r\nSubject: " + subject +
		"\r\n" + content_type + "\r\n\r\n" + body)
	send_to := strings.Split(to, ";")
	err := smtp.SendMail(host, auth, user, send_to, msg)
	return err
}
