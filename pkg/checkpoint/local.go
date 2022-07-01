package checkpoint

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/internal"
	"github.com/gocrane/crane/pkg/metricnaming"
)

var _ Checkpointer = &Local{}

// Local use local filesystem as checkpoint storage, so if you use Local, the craned need some persistent volumes such as cbs as storage to keep the states
type Local struct {
	StoreRoot                   string
	MaxWorkers                  int
	checkpointStoreRequestsChan chan *checkpointStoreRequest
	checkpointLoadRequestsChan  chan *checkpointLoadRequest
	internalReaderFinish        chan struct{}
	internalWriterFinish        chan struct{}
	globalStop                  <-chan struct{}
}

type checkpointStoreRequest struct {
	data *internal.CheckpointContext
}

type checkpointLoadRequest struct {
	namer metricnaming.MetricNamer
	resp  chan *checkpointLoadResponse
}

type checkpointLoadResponse struct {
	data *internal.MetricNamerModelCheckpoint
	err  error
}

func (l *Local) Start(stopCh <-chan struct{}) {
	l.globalStop = stopCh
	writerRoutine := func() {
		for request := range l.checkpointStoreRequestsChan {
			err := l.write(request.data)
			if err != nil {
				klog.ErrorS(err, "Failed to store checkpoint %v", request.data.Namer.BuildUniqueKey())
			}
		}
		l.internalWriterFinish <- struct{}{}
	}

	readerRoutine := func() {
		for request := range l.checkpointLoadRequestsChan {
			data, err := l.read(request.namer)
			if err != nil {
				klog.ErrorS(err, "Failed to load checkpoint %v", request.namer.BuildUniqueKey())
			}
			select {
			case request.resp <- &checkpointLoadResponse{data: data, err: err}:
			}
		}
		l.internalReaderFinish <- struct{}{}
	}
	for i := 0; i < l.MaxWorkers; i++ {
		go writerRoutine()
		go readerRoutine()
	}
}

func SafeCloseStore(ch chan *checkpointStoreRequest) (justClosed bool) {
	defer func() {
		if recover() != nil {
			justClosed = false
		}
	}()

	// assume ch != nil here.
	close(ch)   // panic if ch is closed
	return true // <=> justClosed = true; return
}

func SafeCloseLoad(ch chan *checkpointLoadRequest) (justClosed bool) {
	defer func() {
		if recover() != nil {
			justClosed = false
		}
	}()

	// assume ch != nil here.
	close(ch)   // panic if ch is closed
	return true // <=> justClosed = true; return
}

// Flush flush all pending writer requests to disk, block until all finished
func (l *Local) Flush() {
	SafeCloseLoad(l.checkpointLoadRequestsChan)
	SafeCloseStore(l.checkpointStoreRequestsChan)
	<-l.internalReaderFinish
	<-l.internalWriterFinish
	klog.V(4).Infof("Flush all checkpoint requests")
}

func (l *Local) write(ctx *internal.CheckpointContext) error {
	bytes, err := json.Marshal(ctx.Data)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(l.checkPointFileName(ctx.Namer), bytes, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func (l *Local) read(namer metricnaming.MetricNamer) (*internal.MetricNamerModelCheckpoint, error) {
	bytes, err := ioutil.ReadFile(l.checkPointFileName(namer))
	if err != nil {
		return nil, err
	}
	data := &internal.MetricNamerModelCheckpoint{}
	err = json.Unmarshal(bytes, data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (l *Local) checkPointFileName(namer metricnaming.MetricNamer) string {
	// use md5 to shorten the namer key to avoid file name is too long. each unique metric namer is unique file name, so it is safe.
	// but we can not recover the unique key from md5 vice versa(it do not impact now, because we do not need it), maybe we can find a better compress and decompress algorithms to do this.
	key := namer.BuildUniqueKey()
	encoded := fmt.Sprintf("%x", md5.Sum([]byte(key)))
	return filepath.Join(l.StoreRoot, encoded)
}

func (l *Local) StoreMetricModelCheckpoint(ctx context.Context, checkpoint *internal.CheckpointContext, now time.Time) error {
	return l.write(checkpoint)
}

func (l *Local) AsyncStoreMetricModelCheckpoint(ctx context.Context, checkpoint *internal.CheckpointContext, now time.Time) error {
	defer runtime.HandleCrash()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case l.checkpointStoreRequestsChan <- &checkpointStoreRequest{data: checkpoint}:
		return nil
	}
}

func (l *Local) LoadMetricModelCheckpoint(ctx context.Context, namer metricnaming.MetricNamer) (*internal.MetricNamerModelCheckpoint, error) {
	respChan := make(chan *checkpointLoadResponse, 1)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case l.checkpointLoadRequestsChan <- &checkpointLoadRequest{namer: namer, resp: respChan}:
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-respChan:
		return resp.data, resp.err
	}
}

func NewLocal(config interface{}) (Checkpointer, error) {
	cfg, ok := config.(*LocalStoreConfig)
	if !ok {
		return nil, fmt.Errorf("config type must be *LocalStoreConfig")
	}
	err := os.MkdirAll(cfg.Root, os.ModePerm)
	return &Local{
		internalWriterFinish:        make(chan struct{}, 1),
		internalReaderFinish:        make(chan struct{}, 1),
		StoreRoot:                   cfg.Root,
		MaxWorkers:                  cfg.MaxWorkers,
		checkpointLoadRequestsChan:  make(chan *checkpointLoadRequest, 32),
		checkpointStoreRequestsChan: make(chan *checkpointStoreRequest, 32),
	}, err
}

type LocalStoreConfig struct {
	Root       string
	MaxWorkers int
}

func init() {
	RegisterFactory(StoreTypeLocal, NewLocal)
}
