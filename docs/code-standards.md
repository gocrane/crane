This doc describes the code standards and suggestion for crane project, mainly for new contributor of the project
### import need to be organized
import should be categorized with blank line as system imports, community imports and crane apis and crane imports, like the following example
```
import (
	"reflect"
	"sync"
	"time"

	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/recommender/util"
	
	"github.com/gocrane/api/prediction/v1alpha1"
	
	"github.com/gocrane/crane/pkg/utils"
	"github.com/gocrane/crane/pkg/prediction/config"
)
```

### logs standard
- logs are required for troubleshooting purpose
- log message should always start with capital letter
- log message should be a complete sentence that contains enough context, for example: object key, action, parameters, status, error message
- by default, you don't need to set log level
- set 4 for debug level.
- set 6 for more detail debug level.
- set 10 for massive data log level.
- can use klog.KObj() to contain object key to let we know which object the message is printed for
```go
klog.Infof("Failed to setup webhook %s", "value")
klog.V(4).Infof("Debug info %s", "value")
klog.Errorf("Failed to get scale, ehpa %s error %v", klog.KObj(ehpa), err)
klog.Error(error)
klog.ErrorDepth(5, fmt.Errorf("failed to get ehpa %s: %v", klog.KObj(ehpa), err))
```

### event is needed for critical reconcile loop
- event is to let user know what happens on serverside, only print info we want user to know
- consider failure paths and success paths
- event do not need the object key
```go
c.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedGetSubstitute", err.Error())
```

### comment
- every interface should have comments to clarify 
- comment should be a complete sentence 
```go
// Interface is a source of monitoring metric that provides metrics that can be used for
// prediction, such as 'cpu usage', 'memory footprint', 'request per second (qps)', etc.
type Interface interface {
	// GetTimeSeries returns the metric time series that meet the given
	// conditions from the specified time range.
	GetTimeSeries(metricName string, Conditions []common.QueryCondition,
		startTime time.Time, endTime time.Time, step time.Duration) ([]*common.TimeSeries, error)

	// GetLatestTimeSeries returns the latest metric values that meet the given conditions.
	GetLatestTimeSeries(metricName string, Conditions []common.QueryCondition) ([]*common.TimeSeries, error)

	// QueryTimeSeries returns the time series based on a promql like query string.
	QueryTimeSeries(queryExpr string, startTime time.Time, endTime time.Time, step time.Duration) ([]*common.TimeSeries, error)

	// QueryLatestTimeSeries returns the latest metric values that meet the given query.
	QueryLatestTimeSeries(queryExpr string) ([]*common.TimeSeries, error)
}
```

### functions
- function name should clarify what do this function do, for example: verb + noun
- similar functions should be refactored, merge or divide them
- common functions should move to common folder like utils

### variable
- variable name should clarify what do this variable does, better not use too short name and too simple name
- better to use more meaningful variable name for tmp variable, for example: foo loop

### folder and file
- folder name should be letter with lower case and number
- file name should be letter and number and _

### unit test
- Test-driven developing
- Complex function that include condition decide should add unit test for it

### don't forget to run `make fmt` before you submit code