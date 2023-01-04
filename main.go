package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/chromedp/cdproto/network"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

var (
	URL   string
	Max   int64
	Start int64
)

func init() {
	URL = os.Args[1]
	start, err := strconv.ParseInt(os.Args[2], 10, 64)
	if err != nil {
		panic(err)
	}
	end, err := strconv.ParseInt(os.Args[3], 10, 64)
	if err != nil {
		panic(err)
	}
	if err != nil {
		panic(err)
	}
	if end < start {
		panic(errors.New("Số bắt đầu không thể lớn hơn số "))
	}
	Max = end
	Start = start
	log.Println(os.Args[3], os.Args[1], os.Args[2], URL, Max, Start)
}

func main() {
	for i := Start; i < Max; i++ {
		log.Println("Crawling ", i)
		GetChap(fmt.Sprintf("%v", i), URL)
	}
}

type File struct {
	url      string
	fileName string
	format   string
	page     string
	mangaId  string
	chapter  string
	id       network.RequestID
}

func ResolveURL(url string) string {
	if !strings.HasSuffix(url, ".html") {
		return url + ".html"
	}
	return url
}

func TruyenChapterResolver(url string) string {
	newUrl := strings.Split(url, "/")
	newUrl = newUrl[(len(newUrl) - 1):]
	newUrl = strings.Split(strings.Join(newUrl, ""), "-")
	newUrl = strings.Split(newUrl[len(newUrl)-1], ".")
	return newUrl[0]
}

func URLResolver(url string) ([]string, []string) {
	//https://truyenvua.com/9447/232/0.jpg?gf=hdfgdfg%!

	cleanedURl := strings.Split(url, "?")
	cleanedURl1 := strings.TrimPrefix(cleanedURl[0], "https://truyenvua.com/")
	datas := strings.Split(cleanedURl1, "/")
	datas = datas[len(datas)-3 : len(datas)]
	fileName := datas[len(datas)-1]
	fileFormat := strings.Split(fileName, ".")
	return datas, fileFormat
}

func GetChap(num string, url string) {

	ctx, cancel := chromedp.NewContext(context.Background(), chromedp.WithLogf(log.Printf))
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	done := make(chan bool)

	urlstr := ResolveURL(fmt.Sprintf(strings.TrimSuffix(url, ".html")+"-chap-%v.html", num))
	chap := TruyenChapterResolver(urlstr)

	log.Println("Crawling ", urlstr)

	var requestID = make(map[string]File, 30)
	var ID network.RequestID

	chromedp.ListenTarget(ctx, func(v interface{}) {
		switch ev := v.(type) {
		case *network.EventRequestWillBeSent:
			if strings.Contains(ev.Request.URL, chap) && !strings.Contains(ev.Request.URL, "google") && (strings.Contains(ev.Request.URL, "jpg") || strings.Contains(ev.Request.URL, "png") || strings.Contains(ev.Request.URL, "jpeg")) {
				datas, format := URLResolver(ev.Request.URL)
				if len(format) == 2 && (format[1] == "png" || format[1] == "jpg") {
					log.Println(ev.Request.URL, format[0], format[1], datas[0], datas[1])
					requestID[string(ev.RequestID)] = File{
						url:     ev.Request.URL,
						id:      ev.RequestID,
						page:    format[0],
						format:  format[1],
						mangaId: datas[0],
						chapter: datas[1],
					}
				}
				ID = ev.RequestID
			}
		case *network.EventLoadingFinished:
			if ev.RequestID == ID {
				close(done)
			}
		}
	})

	if err := chromedp.Run(ctx,
		chromedp.Navigate(urlstr),
	); err != nil {
		log.Fatal(err)
	}

	<-done
	// get the downloaded bytes for the request id
	for _, v := range requestID {
		var buf []byte
		if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			buf, err = network.GetResponseBody(v.id).Do(ctx)
			return err
		})); err != nil {
			log.Fatal(err)
		}

		name := "./" + fmt.Sprintf("truyen-%v-chap-%v", v.mangaId, v.chapter) + "/" + fmt.Sprintf("%v.png", v.page)

		os.Mkdir(fmt.Sprintf("truyen-%v-chap-%v", v.mangaId, v.chapter), os.ModePerm)

		if err := os.WriteFile(name, buf, 0644); err != nil {
			log.Fatal(err)
		}
		log.Print("wrote ", name)
	}
}
