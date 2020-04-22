package request

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/gopub/gox"
	"github.com/gopub/wine/mime"
)

type ParamsParser struct {
	maxMemory gox.ByteUnit
}

func NewParamsParser(maxMemory gox.ByteUnit) *ParamsParser {
	p := &ParamsParser{
		maxMemory: maxMemory,
	}
	if p.maxMemory < gox.MB {
		p.maxMemory = gox.MB
	}
	return p
}

func (p *ParamsParser) Parse(req *http.Request) (gox.M, []byte, error) {
	params := gox.M{}
	params.AddMap(p.parseCookie(req))
	params.AddMap(p.parseHeader(req))
	params.AddMap(p.parseURLValues(req.URL.Query()))
	bp, body, err := p.parseBody(req)
	if err != nil {
		return params, body, fmt.Errorf("parse body: %w", err)
	}
	params.AddMap(bp)
	return params, body, nil
}

func (p *ParamsParser) parseCookie(req *http.Request) gox.M {
	params := gox.M{}
	for _, cookie := range req.Cookies() {
		params[cookie.Name] = cookie.Value
	}
	return params
}

func (p *ParamsParser) parseHeader(req *http.Request) gox.M {
	params := gox.M{}
	for k, v := range req.Header {
		k = strings.ToLower(k)
		if strings.HasPrefix(k, "x-") {
			k = k[2:]
			k = strings.Replace(k, "-", "_", -1)
			params[k] = v
		}
	}
	return params
}

func (p *ParamsParser) parseURLValues(values url.Values) gox.M {
	m := gox.M{}
	for k, v := range values {
		i := strings.Index(k, "[]")
		if i >= 0 && i == len(k)-2 {
			k = k[0 : len(k)-2]
			if len(v) == 1 {
				v = strings.Split(v[0], ",")
			}
		}
		k = strings.ToLower(k)
		if len(v) > 1 || i >= 0 {
			m[k] = v
		} else if len(v) == 1 {
			m[k] = v[0]
		}
	}

	return m
}

func (p *ParamsParser) parseBody(req *http.Request) (gox.M, []byte, error) {
	typ := mime.GetContentType(req.Header)
	params := gox.M{}
	switch typ {
	case mime.HTML, mime.Plain:
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return params, nil, fmt.Errorf("read html or plain body: %w", err)
		}
		return params, body, nil
	case mime.JSON:
		body, err := ioutil.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return params, nil, fmt.Errorf("read json body: %w", err)
		}
		if len(body) == 0 {
			return params, nil, nil
		}
		decoder := json.NewDecoder(bytes.NewBuffer(body))
		decoder.UseNumber()
		err = decoder.Decode(&params)
		if err != nil {
			var obj interface{}
			err = json.Unmarshal(body, &obj)
			if err != nil {
				return params, body, fmt.Errorf("decode json failed %s: %w", string(body), err)
			}
		}
		return params, body, nil
	case mime.FormURLEncoded:
		// TODO: will crash
		//body, err := req.GetBody()
		//if err != nil {
		//	return params, nil, fmt.Errorf("get body: %w", err)
		//}
		//bodyData, err := ioutil.ReadAll(body)
		//body.Close()
		//if err != nil {
		//	return params, nil, fmt.Errorf("read form body: %w", err)
		//}
		if err := req.ParseForm(); err != nil {
			return params, nil, fmt.Errorf("parse form: %w", err)
		}
		return p.parseURLValues(req.Form), nil, nil
	case mime.FormData:
		err := req.ParseMultipartForm(int64(p.maxMemory))
		if err != nil {
			return nil, nil, fmt.Errorf("parse multipart form: %w", err)
		}

		if req.MultipartForm != nil && req.MultipartForm.File != nil {
			return p.parseURLValues(req.MultipartForm.Value), nil, nil
		}
		return params, nil, nil
	default:
		body, err := ioutil.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return params, nil, fmt.Errorf("read json body: %w", err)
		}
		return params, body, nil
	}
}
