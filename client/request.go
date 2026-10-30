package client

import (
	"compress/gzip"
	"errors"
	"github.com/Pastir/footballdata/logs"
	"github.com/google/brotli/go/cbrotli"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"time"
)

func RequestToEndpoint(url string) (body []byte, err error) {

	client := http.Client{
		Timeout: 60 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logs.Logs["error"].Error("constructor request: ",
			zap.String("ts", time.Now().String()),
			zap.String("Request", url),
			zap.Error(err),
		)
		return []byte(""), errors.New("400")
	}

	req.Header.Set("accept-encoding", "gzip, br")

	var res *http.Response
	countLoop := 0

	for {
		res, err = client.Do(req)
		// Ok, break
		if res.StatusCode == 200 {
			logs.Logs["access"].Info("new request: ",
				zap.String("ts", time.Now().String()),
				zap.String("Request", url),
				zap.String("status", res.Status),
				zap.Error(err),
			)
			break
		}

		// Error code "429 Too Many Requests" sleep 1 minute
		if res.StatusCode == 429 {
			time.Sleep(time.Duration(60) * time.Second)
			logs.Logs["access"].Info("new request: ",
				zap.String("ts", time.Now().String()),
				zap.String("Request", url),
				zap.String("status", res.Status),
				zap.Error(err),
			)
		}

		if res.StatusCode != 429 && res.StatusCode != 200 && err != nil {
			logs.Logs["error"].Error("err request: ",
				zap.String("ts", time.Now().String()),
				zap.String("Request", url),
				zap.Error(err),
			)
			return []byte(""), err
		}

		// number of requests with error code 429 is limited
		countLoop++
		if countLoop > 10 {
			logs.Logs["error"].Error("err request: ",
				zap.String("ts", time.Now().String()),
				zap.String("Request", url),
				zap.Error(errors.New("to many loops request to endpoint")),
			)
			return []byte(""), err
		}
	}

	defer res.Body.Close()

	switch res.Header.Get("Content-Encoding") {
	case "br":
		reader := cbrotli.NewReader(res.Body)
		defer reader.Close()
		return ioutil.ReadAll(reader)

	case "gzip":
		reader, _ := gzip.NewReader(res.Body)
		defer reader.Close()
		return ioutil.ReadAll(reader)

	default:
		return ioutil.ReadAll(res.Body)
	}
}

func CustomRequest(url string, headers map[string]string) (body []byte, err error) {

	client := http.Client{
		Timeout: 60 * time.Second,
	}

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("accept-encoding", "gzip, br")

	//Установка нужных хидеров
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	res, _ := client.Do(req)
	if res.StatusCode != 200 {
		logs.Logs["error"].Error("response status error: ",
			zap.String("Request", url),
			zap.String("status", res.Status),
		)
		return []byte(""), errors.New("404")
	} else {
		logs.Logs["error"].Info("new request: ",
			zap.String("Request", url),
			zap.String("status", res.Status),
		)
	}

	defer res.Body.Close()

	switch res.Header.Get("Content-Encoding") {
	case "br":
		reader := cbrotli.NewReader(res.Body)
		defer reader.Close()
		return ioutil.ReadAll(reader)

	case "gzip":
		reader, _ := gzip.NewReader(res.Body)
		defer reader.Close()
		return ioutil.ReadAll(reader)

	default:
		return ioutil.ReadAll(res.Body)
	}
}
