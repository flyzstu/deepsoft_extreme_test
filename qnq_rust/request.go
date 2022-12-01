package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type Data struct {
	KnowledgePoints     []string `json:"knowledge"`
	OrigKnowledgePoints []string `json:"orig_knowledge_points"`
	ProcessedText       string   `json:"processed_text"`
	QuestionID          string   `json:"question_id"`
	QuestionText        string   `json:"question_text"`
	Score               float64  `json:"score"`
}

type R struct {
	Code int64  `json:"code"`
	Data []Data `json:"data"`
	Msg  string `json:"msg"`
}

type Q struct {
	Id                  string   `csv:"题目ID"`
	Code                int      `csv:"返回状态码"`
	Subject             string   `csv:"题目"`
	Grade               int      `csv:"年级"`
	KnowledgePoints     []string `csv:"知识点"`
	OrigKnowledgePoints []string `csv:"原知识点"`
	ProcessedText       string   `csv:"相似题目"`
	Cast                float64  `csv:"花费时间"`
}

func request(q *Q, useBackend bool) (error, *Q) {
	var (
		ret   R
		req   *http.Request
		query url.Values
	)
	if useBackend {
		req, _ = http.NewRequest("GET", "http://192.168.31.5:1107/search/api/v1.0/questions", nil)
		query = req.URL.Query()
		query.Add("text", q.Subject)
	} else {
		req, _ = http.NewRequest("GET", "http://192.168.31.5:7085/api/1.0/algorithm/getSearchResultMath", nil)
		query = req.URL.Query()
		query.Add("subject", q.Subject)
		query.Add("account", "1020109")
	}

	req.URL.RawQuery = query.Encode()
	t1 := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Error(err.Error())
		return err, nil
	}
	if resp.StatusCode != http.StatusOK {
		logger.Error("请求失败, 状态码", resp.StatusCode)
		return err, nil
	}
	s, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error(err.Error())
		return err, nil
	}
	err = json.Unmarshal([]byte(s), &ret)
	if err != nil {
		logger.Error(err.Error())
		return err, nil
	}
	if len(ret.Data) < 1 {
		logger.Error("返回的DATA为空, id: %s, msg: %s, 程序返回", q.Id, ret.Msg)
		q.KnowledgePoints = []string{}
		q.ProcessedText = ""
		return nil, q
	}
	q.Cast = time.Since(t1).Seconds()
	q.Code = int(ret.Code)
	q.KnowledgePoints = ret.Data[0].KnowledgePoints
	q.OrigKnowledgePoints = ret.Data[0].OrigKnowledgePoints
	q.ProcessedText = ret.Data[0].ProcessedText
	return nil, q
}
