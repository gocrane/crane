package checkpoint

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gocrane/crane/pkg/internal"
	"github.com/gocrane/crane/pkg/metricnaming"
)

// Checkpointer is used to do checkpoint for metric namer. this package is only responsible for executing store and load of checkpoint data.
// You can implement other checkpoint writer and reader backed by different storages such as localfs、s3、etcd(Custom Resource Definition)
// the caller decides when to do checkpoint, checkpoint frequency is depending on the caller.
// there are multiple ways to decide when to do checkpoint.
// 1. predictor checkpoints all metric namers together periodically by a independent routine. but this will not guarantee the checkpoint data is consistent with the latest updated model in memory
// 2. predictor checkpoints the metric namer each time after model is updating, so the checkpoint is always latest. for example, the percentile to do checkpoint after add sample for each metric namer.
// 3. application caller such as evpa triggers the metric namer to do checkpoint. delegate the trigger to application caller
type Checkpointer interface {
	Start(stopCh <-chan struct{})
	Writer
	Reader
}

type Writer interface {
	// store metricNamer checkpoints. each time call will override original checkpoint data of the same metric namer if it exists.
	// each metric namer model checkpoint only store one replica.
	// this is sync way, it block until the checkpoint stored operation finished
	StoreMetricModelCheckpoint(ctx context.Context, checkpoint *internal.CheckpointContext, now time.Time) error
	// this is async way, it send the checkpoint to a channel and return immediately
	AsyncStoreMetricModelCheckpoint(ctx context.Context, checkpoint *internal.CheckpointContext, now time.Time) error
	// close checkpointer, close the queue && wait until all requests pending in queue done
	Flush()
}

type Reader interface {
	// load metricNamer checkpoints
	LoadMetricModelCheckpoint(ctx context.Context, namer metricnaming.MetricNamer) (*internal.MetricNamerModelCheckpoint, error)
}

type StoreType string

const (
	StoreTypeLocal StoreType = "local"
	StoreTypeK8s   StoreType = "k8s"
)

type Factory func(cfg interface{}) (Checkpointer, error)

var (
	checkpointFactorys = make(map[StoreType]Factory)
	lock               sync.Mutex
)

func RegisterFactory(storeType StoreType, factory Factory) {
	lock.Lock()
	defer lock.Unlock()
	checkpointFactorys[storeType] = factory
}

func InitCheckpointer(storeType StoreType, cfg interface{}) (Checkpointer, error) {
	lock.Lock()
	defer lock.Unlock()
	if factory, ok := checkpointFactorys[storeType]; ok {
		return factory(cfg)
	} else {
		return nil, fmt.Errorf("not registered checkpoint store type %v", storeType)
	}
}
