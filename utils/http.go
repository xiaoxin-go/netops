package utils

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
)

func NewHttpHandler(host string) *HttpHandler {
	h := &HttpHandler{host: host}
	h.init()
	return h
}

type HttpHandler struct {
	host    string
	request *http.Request
	Err     error
}

func (h *HttpHandler) init() {
	if h.request == nil {
		h.request = &http.Request{
			Header:     make(http.Header),
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
		}
	}
}

// 把请求参数转换为buffer
func (h *HttpHandler) getBuffer(body any) (*bytes.Buffer, error) {
	var buffer *bytes.Buffer
	switch {
	case body == nil:
		buffer = bytes.NewBuffer([]byte{})
	case body != nil:
		ve := reflect.ValueOf(body).Elem()
		if ve.Type().String() == "bytes.Buffer" {
			buffer = body.(*bytes.Buffer)
		} else {
			// 把请求体转换成buffer
			bodyBytes, err := json.Marshal(body)
			if err != nil {
				return nil, err
			}
			buffer = bytes.NewBuffer(bodyBytes)
		}
	}
	return buffer, nil
}

// 处理body, 把任意类型的body转换为io.reader
func (h *HttpHandler) setRequestBody(body any) error {
	buffer, err := h.getBuffer(body)
	if err != nil {
		return err
	}
	rc, ok := body.(io.ReadCloser)
	if !ok && body != nil {
		rc = io.NopCloser(buffer)
	}
	h.request.Body = rc
	return nil
}

// 拼接url和host
func (h *HttpHandler) parseUri(uri string) string {
	if !strings.HasPrefix(uri, "http") {
		uri = fmt.Sprintf("%s%s", h.host, uri)
	}
	return uri
}

// SetHeaders 设置请求头
func (h *HttpHandler) SetHeaders(headers map[string]string) {
	for key, value := range headers {
		if key == "Host" {
			h.request.Host = value
		} else {
			h.request.Header.Set(key, value)
		}
	}
}

// SetBasicAuth 设置basicAuth
func (h *HttpHandler) SetBasicAuth(username, password string) {
	h.request.SetBasicAuth(username, password)
}

// 初始化request, 初始化client, 设置headers, 发送请求
func (h *HttpHandler) send(method, uri string, body any, result any) error {
	var (
		res *http.Response
		err error
	)
	u, err := url.Parse(h.parseUri(uri))
	if err != nil {
		return err
	}
	h.request.Method = method
	h.request.Host = u.Host
	h.request.URL = u
	if e := h.setRequestBody(body); e != nil {
		return fmt.Errorf("解析body异常, err: %w", e)
	}

	// 初始化client，并发送请求
	client := http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}}
	// 调用后台接口，重试3次
	for i := 0; i < 3; i++ {
		res, err = client.Do(h.request)
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		return fmt.Errorf("调用接口异常, uri: %s, err: %w", uri, err)
	}
	// 读取数据，转换结果
	bs, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("读取响应结果异常, err: %w", err)
	}
	if res.StatusCode != 200 && res.StatusCode != 204 && res.StatusCode != 201 {
		return fmt.Errorf("请求失败, uri: %s, status_code: %d, text: %s", uri, res.StatusCode, string(bs))
	}
	if result == nil {
		return nil
	}
	if len(bs) > 0 {
		switch result.(type) {
		case *[]byte:
			*result.(*[]byte) = bs
		default:
			if e := json.Unmarshal(bs, result); e != nil {
				return fmt.Errorf("读取结果转换异常, err: %w", e)
			}
		}
	}
	return nil
}
func (h *HttpHandler) Get(uri string, params map[string]string, result any) error {
	if params != nil {
		v := url.Values{}
		for k, value := range params {
			v.Add(k, value)
		}
		urlParams := v.Encode()
		uri = fmt.Sprintf("%s?%s", uri, urlParams)
	}
	return h.send(http.MethodGet, uri, nil, result)
}
func (h *HttpHandler) Post(uri string, body any, result any) error {
	return h.send(http.MethodPost, uri, body, result)
}
func (h *HttpHandler) Put(uri string, body any, result any) error {
	return h.send(http.MethodPut, uri, body, result)
}
