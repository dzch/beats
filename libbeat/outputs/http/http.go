package http

import (
	"expvar"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
)

type transData struct {
	signaler op.Signaler
	opts     outputs.Options
	event    common.MapStr
}

type httpOut struct {
	done          chan struct{}
	transDataChan chan *transData
}

var debugf = logp.MakeDebug("http")

// Metrics that can retrieved through the expvar web interface.
var (
	statReadBytes   = expvar.NewInt("libbeat.http.publish.read_bytes")
	statWriteBytes  = expvar.NewInt("libbeat.http.publish.write_bytes")
	statReadErrors  = expvar.NewInt("libbeat.http.publish.read_errors")
	statWriteErrors = expvar.NewInt("libbeat.http.publish.write_errors")
)

const (
	defaultWaitRetry    = 1 * time.Second
	defaultMaxWaitRetry = 60 * time.Second
)

func init() {
	outputs.RegisterOutputPlugin("http", new)
}

func new(beatName string, cfg *common.Config, expireTopo int) (outputs.Outputer, error) {
	r := &httpOut{}
	if err := r.init(cfg, expireTopo); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *httpOut) init(cfg *common.Config, expireTopo int) error {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return err
	}
	r.done = make(chan struct{})
	r.transDataChan = make(chan *transData, config.QueueSize)
	for i := 0; i < config.WorkerNum; i++ {
		hw := &httpWorker{
			config:        &config,
			id:            i,
			done:          r.done,
			transDataChan: r.transDataChan,
		}
		err := hw.init()
		if err != nil {
			return err
		}
		go hw.run()
	}
	return nil
}

func (r *httpOut) Close() error {
	close(r.done)
	return nil
}

func (r *httpOut) PublishEvent(
	signaler op.Signaler,
	opts outputs.Options,
	data outputs.Data,
	/*event common.MapStr,*/
) error {
	// TODO: using pool
	td := &transData{
		signaler: signaler,
		opts:     opts,
		event:    data.Event,
		/*event:    event,*/
	}
	r.transDataChan <- td
	return nil
}

//func (r *httpOut) BulkPublish(
//	signaler op.Signaler,
//	opts outputs.Options,
//	events []common.MapStr,
//) error {
//	return r.mode.PublishEvents(signaler, opts, events)
//}
