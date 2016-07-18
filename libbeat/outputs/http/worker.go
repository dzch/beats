package http

import (
	"bytes"
	"fmt"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"io"
	"net"
	"net/http"
	"time"
)

type httpWorker struct {
	config        *httpConfig
	id            int
	done          <-chan struct{}
	transDataChan <-chan *transData
	client        *http.Client
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
	msgi, ok := td.event["message"]
	if !ok {
		logp.Err("unsupported event: message is not set, skip: %#v", td.event)
		return
	}
	msg := msgi.(*string)
	buf := bytes.NewReader([]byte(*msg))
sendloop:
	for {
		select {
		case <-h.done:
			return
		default:
		}
		buf.Seek(0, 0)
		err := h.doPost(buf)
		if err == nil {
			op.SigCompleted(td.signaler)
			break sendloop
		}
		// fail retry
		logp.Err("http worker [%d] fail to send data: %s, %s", h.id, h.config.Url, err.Error())
		sendTimes++
		if h.config.MaxRetries == -1 || sendTimes < h.config.MaxRetries+1 {
			time.Sleep(h.config.FailRetryInterval)
		} else {
			// reachead max retires
			op.SigCompleted(td.signaler)
		}
	}
}

func (h *httpWorker) doPost(data io.Reader) error {
	rsp, err := h.client.Post(
		h.config.Url,
		"application/octet-stream",
		data)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != 200 {
		return fmt.Errorf("response status code %d", rsp.StatusCode)
	}
	return nil
}
