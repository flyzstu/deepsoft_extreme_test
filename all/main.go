package main

import (
	"bytes"
	"flag"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/gocarina/gocsv"
	"github.com/goccy/go-json"

	"github.com/flyzstu/mylog"
)

type ImageINFO struct {
	Subject    string
	ID         int
	FileName   string
	UploadName string
	Flag       int
}

type Response struct {
	ID           int           `csv:"id"`
	Cast         float32       `csv:"cast"`
	Msg          string        `csv:"msg" json:"msg"`
	Code         int64         `csv:"code" json:"code"`
	Data         []interface{} `csv:"data" json:"data"`
	BusinessCode int64         `csv:"businessCode" json:"businessCode"`
}

type ImageObj struct {
	ID          int
	Type        int
	ImageBuf    *bytes.Buffer
	ContentType string
	FileName    string
	Account     string
	Grade       string
}

var (
	logger       = mylog.New()
	ImageOBJChan = make(chan *ImageObj, 10000)
	RespOBJChan  = make(chan *Response, 10000)
	// ImageINFOChan  = make(chan *ImageINFO, 10000)
	// ImageOBJChan   = make(chan *ImageObj, 10000)
	// FingerRespChan = make(chan *FingerCSV, 10000)
	// PaddleRespChan = make(chan *PaddleCSV, 10000)
	// FingerCSVChan  = make(chan *FingerCSV, 10000)

	FinishedChan1 = make(chan struct{}, 10000)
	FinishedChan2 = make(chan struct{}, 10000)
	FinishedChan3 = make(chan struct{}, 10000)
)

func openIMG(filename string, threads int) {
	for i := 0; i < threads; i++ {
		buf := new(bytes.Buffer)
		writer := multipart.NewWriter(buf)
		err := writer.WriteField("type", "2")
		if err != nil {
			logger.Error(err.Error())
			panic(err)
		}
		err = writer.WriteField("account", "1020109")
		if err != nil {
			logger.Error(err.Error())
			panic(err)
		}
		err = writer.WriteField("grade", "4")
		if err != nil {
			logger.Error(err.Error())
			panic(err)
		}
		contentType := writer.FormDataContentType()
		formFile, err := writer.CreateFormFile("file", filename) // 提供表单中的字段名<img>和文件名<new.jpg>,返回值是可写的接口io.Writer
		if err != nil {
			logger.Error(err.Error())
			panic(err)
		}
		srcFile, err := os.Open(filename)
		if err != nil {
			logger.Error(err.Error())
		}

		_, err = io.Copy(formFile, srcFile)
		if err != nil {
			logger.Error("Write to form file falied: %s\n", err.Error())
			panic(err)
		}

		err = writer.Close()
		if err != nil {
			logger.Error("Write to form file falied: %s\n", err.Error())
			panic(err)
		}
		ImageOBJChan <- &ImageObj{
			ID:          i,
			FileName:    filename,
			ImageBuf:    buf,
			ContentType: contentType,
		}
		logger.Info("图片打开成功, ID为：%d", i)

		err = srcFile.Close()
		if err != nil {
			logger.Error("Write to form file falied: %s\n", err.Error())
			panic(err)
		}
	}

}
func finger(threads int) {
	for i := 0; i < threads; i++ {
		go func() {
			for imgobj := range ImageOBJChan {
				respObj := new(Response)
				t1 := time.Now()
				resp, err := http.Post("http://192.168.31.5:7085/api/1.0/algorithm/getSearchResult", imgobj.ContentType, imgobj.ImageBuf)
				if err != nil {
					FinishedChan2 <- struct{}{}
					logger.Error("post failed, err:%v\n", err)
					continue
				}
				respObj.Cast = float32(time.Since(t1).Seconds())
				// data, err := ioutil.ReadAll(resp.Body)
				err = json.NewDecoder(resp.Body).Decode(respObj)
				if err != nil {
					FinishedChan2 <- struct{}{}
					logger.Error("decoder failed, err:%v\n", err)
					continue
				}
				respObj.ID = imgobj.ID

				RespOBJChan <- respObj
				FinishedChan2 <- struct{}{}
				resp.Body.Close()
				logger.Info("getSearchResult请求成功，ID为: %d", respObj.ID)

			}
		}()

	}
}

func saveCSV() {

	fingerFILE, err := os.OpenFile("100线程请求1图片.csv", os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	var (
		fingercsvs = []*Response{}
	)

	for fingercsv := range RespOBJChan {
		logger.Info("添加成功，ID为：%d", fingercsv.ID)
		fingercsvs = append(fingercsvs, fingercsv)
	}

	if err := gocsv.MarshalFile(&fingercsvs, fingerFILE); err != nil {
		logger.Error("MarshalFile error: %s", err.Error())
		return
	}
	defer fingerFILE.Close()

}

func wait(length int) {
	for i := 0; i < length; i++ {
		<-FinishedChan2
	}
	close(RespOBJChan)
}

func main() {
	defer logger.Close()
	var threads = flag.Int("t", 100, "线程数")
	flag.Parse()
	openIMG("WIN_20220622_14_24_50_Pro.jpg", *threads)
	finger(*threads)
	wait(*threads)
	saveCSV()

}
