package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"io"
	"math/rand"
	"net"
	"net/http"
	"regexp"
	"time"
)

type httpWorker struct {
	config        *httpConfig
	id            int
	done          <-chan struct{}
	transDataChan <-chan *transData
	client        *http.Client
	regTopic      *regexp.Regexp
	hasRegTopic   bool
	addrId        int
	addrNum       int
}

func (h *httpWorker) init() error {
	h.client = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   h.config.CTimeout,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		Timeout: h.config.ProcTimeout,
	}
	var err error
	h.regTopic, err = regexp.Compile("{#TOPIC}")
	if err != nil {
		logp.Critical("fail to regexp.Compile: %s", err.Error())
		return err
	}
	h.hasRegTopic = h.regTopic.Match([]byte(h.config.Uri))
	h.addrNum = len(h.config.Addrs)
	h.addrId = rand.Intn(h.addrNum)
	return nil
}

func (h *httpWorker) run() {
	for {
		select {
		case <-h.done:
			return
		case td := <-h.transDataChan:
			h.processTransData(td)
		}
	}
}

func (h *httpWorker) processTransData(td *transData) {
	sendTimes := 0
	//msgi, ok := td.event["message"]
	//if !ok {
	//	logp.Err("unsupported event: message is not set, skip: %#v", td.event)
	//	return
	//}
	//msg := msgi.(*string)
	//buf := bytes.NewReader([]byte(*msg))
	data, err := json.Marshal(td.event)
	if err != nil {
		logp.Err("http worker [%d] Fail to json encode event(%v): %#v", h.id, err, td.event)
		op.SigCompleted(td.signaler)
		return
	}
	buf := bytes.NewReader(data)
sendloop:
	for {
		h.addrId = (h.addrId + 1) % h.addrNum
		select {
		case <-h.done:
			return
		default:
		}
		buf.Seek(0, 0)
		err := h.doPost(td.event, buf)
		if err == nil {
			op.SigCompleted(td.signaler)
			break sendloop
		}
		// fail retry
		logp.Err("http worker [%d] fail to send data: %s", h.id, err.Error())
		sendTimes++
		if h.config.MaxRetries == -1 || sendTimes < h.config.MaxRetries+1 {
			time.Sleep(h.config.FailRetryInterval)
		} else {
			// reachead max retires
			op.SigCompleted(td.signaler)
		}
	}
}

func (h *httpWorker) doPost(event common.MapStr, data io.Reader) error {
	uri := h.config.Uri
	if h.hasRegTopic {
		fds, ok := event["fields"]
		if !ok {
			return fmt.Errorf("fields is not configured")
		}
		fdm, ok := fds.(common.MapStr)
		if !ok {
			return fmt.Errorf("fields is not map")
		}
		topici, ok := fdm["topic"]
		if !ok {
			return fmt.Errorf("fields does not contain 'topic' item")
		}
		topic, ok := topici.(string)
		if !ok {
			return fmt.Errorf("topic in fields is not string")
		}
		uri = h.regTopic.ReplaceAllString(h.config.Uri, topic)
	}
	url := fmt.Sprintf("http://%s%s", h.config.Addrs[h.addrId], uri)
	rsp, err := h.client.Post(
		url,
		"application/json",
		data)
	if err != nil {
		return fmt.Errorf("url: %s, err: %s", url, err.Error())
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != 200 {
		return fmt.Errorf("url: %s, err: response status code %d", url, rsp.StatusCode)
	}
	return nil
}
