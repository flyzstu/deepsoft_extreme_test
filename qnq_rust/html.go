package main

import (
	"fmt"
	"net/http"
	"unicode"

	"github.com/PuerkitoBio/goquery"
)

// 根据id获取题目
func getQuestions(id string, onlyHan bool) (error, string) {
	URL := "http://192.168.31.5:6789/question/" + id
	res, err := http.Get(URL)
	if err != nil {
		logger.Error(err.Error())
		return err, ""
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		err = fmt.Errorf("id: %s, error: %s", id, res.Status)
		return err, ""
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		logger.Error(err.Error())
		return err, ""
	}
	s := doc.Find("div[class=question_content]")

	var hanChar []rune
	if onlyHan {
		for _, char := range s.Text() {
			if unicode.Is(unicode.Han, char) {
				hanChar = append(hanChar, char)
			}
		}
	} else {
		for _, char := range s.Text() {
			if unicode.Is(unicode.Han, char) || unicode.IsDigit(char) || unicode.IsPunct(char) {
				hanChar = append(hanChar, char)
			}
		}
	}

	return nil, string(hanChar)
}
