package main

import (
	"bytes"
	"flag"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/gocarina/gocsv"
	"github.com/goccy/go-json"

	"github.com/flyzstu/mylog"
)

type ImageINFO struct {
	ID         int
	FileName   string
	UploadName string
	Flag       string
	Coordinate string
}

type ImageObj struct {
	Subject     string
	Flag        string
	ID          int
	ImageBuf    *bytes.Buffer
	ContentType string
	FileName    string
	Coordinate  string
}

type ResponseObject struct {
	Msg          string        `json:"msg"`
	Code         int64         `json:"code"`
	Data         []interface{} `json:"data"`
	BusinessCode int64         `json:"businessCode"`
}

type CSVObject struct {
	Code     int64         `csv:"状态码"`
	Cast     float32       `csv:"响应时间"`
	Response []interface{} `csv:"返回消息"`
	Subject  string        `csv:"科目"`
}

var (
	logger         = mylog.New()
	coordinate     = map[string]string{}
	ImageINFOChan  = make(chan *ImageINFO, 10000)
	ImageOBJChan   = make(chan *ImageObj, 10000)
	CSVObjectpChan = make(chan *CSVObject, 10000)

	FinishedChan1 = make(chan struct{}, 10000)
	FinishedChan2 = make(chan struct{}, 10000)
	FinishedChan3 = make(chan struct{}, 10000)
)

func getPicINFO(dir string, threads int, FLAG string) (length int) {

	fs, err := os.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	for _, file := range fs {
		// if file.Name() == "语文" {
		// 	FLAG = "1"
		// } else if file.Name() == "数学" {
		// 	FLAG = "2"
		// } else if file.Name() == "英语" {
		// 	FLAG = "3"
		// }

		// images, err := os.ReadDir(path.Join(dir, file.Name()))
		// if err != nil {
		// 	panic(err)
		// }

		// for _, img := range images {
		img := file
		for i := 0; i < threads; i++ {
			length++
			if _, ok := coordinate[img.Name()]; !ok {
				logger.Error("找不到指定文件的坐标, 文件名:%s", img.Name())
				continue
			}
			picPATH := path.Join(path.Join(dir, img.Name()))
			// logger.Debug("picPATH: %v", picPATH)
			ImageINFOChan <- &ImageINFO{
				ID:         length,
				FileName:   picPATH,
				Flag:       FLAG,
				UploadName: img.Name(),
				Coordinate: coordinate[img.Name()],
			}
		}
	}
	logger.Info("需要打开%d个文件", length)
	return length
}

func readJson(filename string) error {
	xyz, err := os.ReadFile(filename)
	if err != nil {
		logger.Error("打开坐标文件失败, 错误: %s", err.Error())
		return err
	}
	err = json.Unmarshal(xyz, &coordinate)
	if err != nil {
		logger.Error("反序列化坐标失败, 错误: %s", err.Error())
		return err
	}
	return nil
}

func openIMG(threads int) {
	for i := 0; i < threads; i++ {
		go func() {
			for imageinfo := range ImageINFOChan {
				buf := new(bytes.Buffer)
				writer := multipart.NewWriter(buf)
				err := writer.WriteField("type", imageinfo.Flag)
				if err != nil {
					FinishedChan1 <- struct{}{}
					logger.Error(err.Error())
					continue
				}
				err = writer.WriteField("account", "1096106")
				if err != nil {
					FinishedChan1 <- struct{}{}
					logger.Error(err.Error())
					continue
				}
				err = writer.WriteField("grade", "2")
				if err != nil {
					FinishedChan1 <- struct{}{}
					logger.Error(err.Error())
					continue
				}
				err = writer.WriteField("coordinate", imageinfo.Coordinate)
				if err != nil {
					FinishedChan1 <- struct{}{}
					logger.Error(err.Error())
					continue
				}
				contentType := writer.FormDataContentType()
				formFile, err := writer.CreateFormFile("file", imageinfo.UploadName) // 提供表单中的字段名<img>和文件名<new.jpg>,返回值是可写的接口io.Writer
				if err != nil {
					FinishedChan1 <- struct{}{}
					logger.Error(err.Error())
					continue
				}
				srcFile, err := os.Open(imageinfo.FileName)
				if err != nil {
					FinishedChan1 <- struct{}{}
					logger.Error(err.Error())
					continue
				}
				_, err = io.Copy(formFile, srcFile)
				if err != nil {
					FinishedChan1 <- struct{}{}
					logger.Error("Write to form file falied: %s\n", err.Error())
					continue
				}
				err = srcFile.Close()
				if err != nil {
					FinishedChan1 <- struct{}{}
					logger.Error("Write to form file falied: %s\n", err.Error())
					continue
				}
				err = writer.Close()
				if err != nil {
					FinishedChan1 <- struct{}{}
					logger.Error("Write to form file falied: %s\n", err.Error())
					continue
				}
				ImageOBJChan <- &ImageObj{
					FileName:    imageinfo.UploadName,
					Flag:        imageinfo.Flag,
					ImageBuf:    buf,
					ContentType: contentType,
					Coordinate:  imageinfo.Coordinate,
				}
				logger.Info("图像打开成功，ID为: %d", imageinfo.ID)
				FinishedChan1 <- struct{}{}
			}
		}()
	}
}
func worker(threads int) {
	for i := 0; i < threads; i++ {
		go func() {
			for imgobj := range ImageOBJChan {
				responseObj := new(ResponseObject)
				t1 := time.Now()
				req, err := http.NewRequest("POST", "http://machine.deepsoft-tech.com:8081/study/api/1.0/algorithm/getSearchResultNoCoordinate", imgobj.ImageBuf)
				if err != nil {
					FinishedChan2 <- struct{}{}
					logger.Error("构造请求失败:%s", err.Error())
					continue
				}
				req.Header.Set("token", "eyJ0eXBlIjoiand0IiwiYWxnIjoiSFMyNTYiLCJ0eXAiOiJKV1QifQ.eyJleHAiOjE2Njc1MjcxOTIsImFjY291bnQiOiIwIn0.VPG3FgDPIA3KzZwOwQ8mYSb8D4Sbi8z1vlw2a_Il7og")
				req.Header.Set("Content-Type", imgobj.ContentType)
				// resp, err := http.Post("http://machine.deepsoft-tech.com:8081/study/api/1.0/algorithm/getSearchResultNoCoordinate", imgobj.ContentType, imgobj.ImageBuf)
				// if err != nil {
				// 	FinishedChan2 <- struct{}{}
				// 	logger.Error("post failed, err:%v\n", err)
				// 	continue
				// }
				resp, err := (&http.Client{}).Do(req)
				if err != nil {
					FinishedChan2 <- struct{}{}
					logger.Error("post failed, err:%s", err.Error())
					continue
				}
				t2 := time.Now()
				err = json.NewDecoder(resp.Body).Decode(responseObj)
				if err != nil {
					FinishedChan2 <- struct{}{}
					logger.Error("decoder failed, err:%v\n", err)
					continue
				}
				CSVObjectpChan <- &CSVObject{
					Code:     responseObj.Code,
					Cast:     float32(t2.Sub(t1).Seconds()),
					Response: responseObj.Data,
					Subject:  imgobj.Flag,
				}
				FinishedChan2 <- struct{}{}
				resp.Body.Close()
				logger.Info("请求成功，耗时为 %.2f", float32(t2.Sub(t1).Seconds()))
			}
		}()

	}
}

func saveCSV(filename string) {

	FILE, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	var (
		csvs = []*CSVObject{}
	)

	for csv := range CSVObjectpChan {
		// logger.Info("csv添加成功，耗时为：%s", csv.Cast)
		csvs = append(csvs, csv)
	}

	if err := gocsv.MarshalFile(&csvs, FILE); err != nil {
		panic(err)
	}
	defer FILE.Close()

}

func wait(length int) {
	for i := 0; i < length; i++ {
		<-FinishedChan1
	}
	for i := 0; i < length; i++ {
		<-FinishedChan2
	}
	close(CSVObjectpChan)
}

func main() {
	defer logger.Close()
	err := readJson("xyz.json")
	if err != nil {
		panic(err)
	}
	var threads = flag.Int("t", 1000, "线程数")
	var dir = flag.String("d", "tmp/英语", "文件夹地址")
	flag.Parse()
	L := getPicINFO(*dir, *threads, "3")
	openIMG(*threads)
	worker(*threads)
	wait(L)
	saveCSV("5个搜题实例3个OCR实例1000个线程请求英语.csv")

}
