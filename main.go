package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/chromedp/chromedp/device"
	"github.com/chromedp/chromedp"
	"github.com/panjf2000/ants"
)

var domain_list = make(chan string, 20)
var wg sync.WaitGroup

func readfile(filename string) {
	file, err := os.Open(filename)
	defer file.Close()
	defer close(domain_list)
	if err == nil {
		r := bufio.NewReader(file)
		for {
			domain_b, _, err := r.ReadLine()
			domain := strings.Replace(string(domain_b), " ", "", -1)
			if len(string(domain)) != 0 && err == nil {
				domain_list <- string(domain)
			} else {
				break
			}

		}
	} else {
		fmt.Println("readfile_err: ", err)
	}
}

func getscreen(domain string, savepath string) (bool, string) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()


	if len(domain)==0{
		return false,""
	}

	match, _ := regexp.MatchString("^(http|https)://", domain)
	if !match {
		domain = "http://" + domain
	}
	var b2 []byte
	if err := chromedp.Run(ctx,
		chromedp.Emulate(device.Reset),
		chromedp.EmulateViewport(1920, 920),
		chromedp.Navigate(domain),
		chromedp.CaptureScreenshot(&b2),
	); err != nil {
	}
	if len(b2)==0{
		return false,""
	}
	// 根据domain设置文件名
	domain_res := strings.TrimLeft(domain, "http://")
	domain_res = strings.TrimLeft(domain_res, "https://")
	re3, _ := regexp.Compile("[^A-Za-z0-9\u4e00-\u9fa5]");
	output_filename := re3.ReplaceAllString(domain_res, "_");
	// 如果文件夹不存在则创建
	_, err := os.Stat(savepath)
	if err != nil {
		if os.IsNotExist(err) {
			os.Mkdir(savepath, 0777)
		}
	}
	_, err = os.Stat(savepath+"/img")
	if err != nil {
		if os.IsNotExist(err) {
			os.Mkdir(savepath+"/img", 0777)
		}
	}
	savepath = strings.TrimRight(savepath, "/")
	savepath = strings.TrimRight(savepath, "\\")

	if err := ioutil.WriteFile(savepath+"/img/"+output_filename+".png", b2, 0777); err != nil {
		fmt.Println("2", err)
	}
	return true, output_filename
}


func main() {
	var domain_file string
	var output_path string
	flag.StringVar(&domain_file, "f", "domain.txt", "目标存放文件")
	flag.StringVar(&output_path, "o", "./output", "输出的文件夹")
	flag.Parse()
	p, _ := ants.NewPool(1000)
	os.Mkdir(output_path, 0777)
	md_file, err := os.OpenFile(output_path+"/index.md", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777) //写入的md文件
	if err != nil {
		if os.IsNotExist(err) {
			os.Mkdir(output_path, 0777)
		}
	}
	p.Submit(func() {
		readfile(domain_file)
	})
	for {

		domain, isclose := <-domain_list
		if len(domain) == 0{
			if !isclose && len(domain_list) <= 0 {
				break
			}else{
				continue
			}
		}
		wg.Add(1)
		_ = p.Submit(func() {
			defer wg.Done()
			isok, filename := getscreen(domain, output_path)
			if isok {
				fmt.Println("[+] "+domain)
				md_link := "# [" + domain + "](" + domain + ")"
				md_pic := "![" + filename + ".png](img/" + filename + ".png)"
				md_file.WriteString(md_link + "\n\n\n" + md_pic + "\n")
			}else{
				if len(domain)!=0{
					fmt.Println("[-] "+domain)
				}
			}
		})
		if !isclose && len(domain_list) <= 0 {
			break
		}
	}
	wg.Wait()
	return

}
