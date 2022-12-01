package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	"github.com/flyzstu/mylog"
	"github.com/gocarina/gocsv"
)

type Ret struct {
	Code int64 `json:"code"`
	// Data         string `json:"data"`
	Msg          string `json:"msg"`
	BusinessCode int64  `json:"businessCode"`
}

type question struct {
	Id           string  `csv:"题目ID"`
	Code         int     `csv:"返回状态码"`
	Subject      string  `csv:"题目"`
	Grade        int     `csv:"年级"`
	BusinessCode int64   `csv:"businessCode"`
	Ret          string  `csv:"返回值"`
	Cast         float64 `csv:"花费时间"`
}

var (
	logger           = mylog.New()
	questionIDschan  = make(chan *question, 100000)
	hansStringChan   = make(chan *question, 100000)
	timeChan         = make(chan *question, 100000)
	fininshedChan    = make(chan struct{}, 100000)
	reqfininshedChan = make(chan struct{}, 100000)
)

func getHan(id string) (error, string) {
	URL := "http://192.168.31.5:6789/question/" + id
	res, err := http.Get(URL)
	if err != nil {
		logger.Error(err.Error())
		return err, ""
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		err = fmt.Errorf("id: %s, error: %s", id, res.Status)
		// logger.Error("id: %s, error: %s", id, res.Status)
		return err, ""
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		logger.Error(err.Error())
		return err, ""
	}
	s := doc.Find("div[class=question_content]")

	var hanChar []rune
	for _, char := range s.Text() {
		// if unicode.Is(unicode.Han, char) {
		if unicode.Is(unicode.Han, char) || unicode.IsDigit(char) || unicode.IsPunct(char) {
			hanChar = append(hanChar, char)
		}
	}
	// fmt.Printf("hanChar: %v\n", string(hanChar))
	return nil, string(hanChar)
	// return nil, s.Text()
}

func reqAPI(q *question) (error, *question) {
	var ret Ret
	// http: //192.168.31.167/search/api/v1.0/questions
	// req, err := http.NewRequest("GET", "http://192.168.31.5:7085/api/1.0/algorithm/getSearchResultMath", nil)
	// req, err := http.NewRequest("GET", "http://192.168.31.75:1107/search/api/v1.0/questions", nil)
	req, err := http.NewRequest("GET", "http://machine.deepsoft-tech.com:8081/study/api/1.0/algorithm/getSearchResultMath", nil)
	if err != nil {
		logger.Error(err.Error())
		return err, nil
	}
	req.Header.Add("token", "eyJ0eXBlIjoiand0IiwiYWxnIjoiSFMyNTYiLCJ0eXAiOiJKV1QifQ.eyJleHAiOjE2Njc1MjcxOTIsImFjY291bnQiOiIwIn0.VPG3FgDPIA3KzZwOwQ8mYSb8D4Sbi8z1vlw2a_Il7og")

	query := req.URL.Query()
	// query.Add("subject", q.Subject)
	query.Add("subject", q.Subject)
	query.Add("grade", fmt.Sprint(q.Grade))
	query.Add("account", "1020109")
	req.URL.RawQuery = query.Encode()
	// fmt.Printf("req.URL: %v\n", req.URL)
	t1 := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Error(err.Error())
		return err, nil
	}
	if resp.StatusCode != http.StatusOK {
		logger.Error("resp.StatusCode:", resp.StatusCode)
		return err, nil
	}
	s, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error(err.Error())
		return err, nil
	}
	q.Cast = time.Since(t1).Seconds()
	err = json.Unmarshal([]byte(s), &ret)
	if err != nil {
		logger.Error(err.Error())
		return err, nil
	}
	// if len(ret.Data) < 1 {
	// 	logger.Error("检索不到ret.Data, id: %s, msg: %s, 程序返回", q.Id, ret.Msg)
	// 	return nil, q
	// }

	q.Code = int(ret.Code)
	// q.KnowledgePoints = ret.Data[0].KnowledgePoints
	// q.OrigKnowledgePoints = ret.Data[0].OrigKnowledgePoints
	// q.ProcessedText = ret.Data[0].ProcessedText
	q.Ret = string(s)
	q.BusinessCode = ret.Code

	return nil, q
}

func reqester() {
	for q := range hansStringChan {
		err, q := reqAPI(q)
		if err != nil {
			logger.Error("查询失败，错误：%s", err.Error())
			reqfininshedChan <- struct{}{}
			continue
		}
		timeChan <- q
		reqfininshedChan <- struct{}{}
	}
}

func worker() {
	for q := range questionIDschan {
		err, hanString := getHan(q.Id)
		if err != nil {
			// logger.Error("%s: HTML解析失败，错误：%s", q.Id, err)
			fininshedChan <- struct{}{}
			reqfininshedChan <- struct{}{}
			continue
		}
		q.Subject = hanString
		hansStringChan <- q
		fininshedChan <- struct{}{}
	}
}

func main() {
	defer logger.Close()
	var GradedQids = make(map[string][]string)
	s, err := ioutil.ReadFile("graded_qids.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(s, &GradedQids)
	if err != nil {
		panic(err)
	}

	for i := 0; i < 1000; i++ {
		go worker()
	}
	for i := 0; i < 1000; i++ {
		go reqester()
	}
	t1 := time.Now()
	var count int
	for i := 1; i < 6; i++ {
		questionsID := GradedQids[fmt.Sprintf("%d", i)]
		for j := 0; j < 1000; j++ {
			count++
			q := &question{
				Id:    questionsID[j],
				Grade: i,
			}
			questionIDschan <- q
		}
	}

	for i := 0; i < count; i++ {
		<-fininshedChan
	}
	// fmt.Println("FLAG1")
	for i := 0; i < count; i++ {
		<-reqfininshedChan
	}
	// fmt.Println("FLAG2")
	var qs []*question
	for i := 0; i < count; i++ {
		tmp := <-timeChan
		// logger.Info("index: %d, csv成功插入数据，id为:%s", i, tmp.Id)
		qs = append(qs, tmp)
	}

	logger.Info("请求时间：%f秒", time.Since(t1).Seconds())
	// fmt.Println("FLAG3")
	filename := time.Now().Format("20060102150405")
	clientsFile, err := os.OpenFile(filename+".csv", os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		panic(err)
	}
	clientsFile.WriteString("\xEF\xBB\xBF")

	// csvContent, err := gocsv.MarshalString(&qs)
	// if err != nil {
	// 	panic(err)
	// }

	err = gocsv.MarshalFile(&qs, clientsFile)
	if err != nil {
		panic(err)
	}
	// fmt.Println(csvContent) // Display all clients as CSV string
	logger.Info("OK")

}
