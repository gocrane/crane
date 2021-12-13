This doc describes the code standards and suggestion for crane project, mainly for new contributor of the project
### import need to be organized
import should be categorized with blank line as system imports, community imports and crane imports, like the following example
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
- by default, you don't need to set log level
- set 4 or bigger number for finer debug level.
- log should contain object key so from logs we know which object the message is printed for
```
log.Logger().Info("Failed to setup webhook", "key","value")
log.Logger().V(4).Info("Debug info", "key","value")
log.Error(err, "Failed to get HPA", "ehpa", klog.KObj(ehpa))
```
### event is needed for critical reconcile loop
- event is to let user know what happens on serverside, only print info we want user to know
- consider failure paths and success paths
- event do not need the object key
```
c.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedGetSubstitute", err.Error())
```
### don't forget to run `make fmt` before you submit code