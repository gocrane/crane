---
title: How to print log in crane
authors:
- "@devinyan"
  reviewers:
- "@AAA"
  creation-date: 2021-11-18
  last-updated: 2021-11-18
  status: implementable
---

# Title
- How to print log in crane


# Log frame work init
In the main function, we can do this action to init the log 

```
import (
	"fmt"
	"os"

	"k8s.io/component-base/logs"
	ctrl "sigs.k8s.io/controller-runtime"
	"github.com/gocrane/crane/pkg/utils/log"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	clogs.InitLogs("craned")
}
```

# Log use basic log

```
import (
   "github.com/gocrane/crane/pkg/utils/log"
)

func A() {
   log.Logger().V(2).Info("run manager")
   log.Logger().Error(err, "opts validate failed")
}

```

# Log with name

```
import (
   "github.com/gocrane/crane/pkg/utils/log"
)

func A() {
   log.Logger().WithName("extent-name").Info("run manager")
   log.Logger().WithName("extent-name").Error(err, "opts validate failed")
}

```

when in controller, we can initialize a logger when new the controller manager
```
&xxxx.xxxxxController{
		Client:     mgr.GetClient(),
		Log:        log.Logger().WithName("extent-name"),
		Scheme:     mgr.GetScheme(),
	}
```

then in the controller logics to use the log like this(p is the ptr of the controller manager):
```
   p.Log().Info("run controller")
   p.Log().Error(err, "conroller failed")
```

# Log with object

we can use `GenerateKey` to print the info of resource object, like this:
```
log.Logger().V(2).Info("object %s is updated successfully", clogs.GenerateKey(object.Name, object.Namespace))
```