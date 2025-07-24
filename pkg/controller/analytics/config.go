package analytics

/*
import (
	"github.com/fsnotify/fsnotify"
	"github.com/gocrane/crane/pkg/recommend"
	"k8s.io/klog/v2"
)

func (c *Controller) watchConfigSetFile() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		klog.Error(err)
		return
	}
	defer watcher.Close()

	err = watcher.Add(c.ConfigSetFile)
	if err != nil {
		klog.ErrorS(err, "Failed to watch", "file", c.ConfigSetFile)
		return
	}
	klog.Infof("Start watching %s for update.", c.ConfigSetFile)

	for {
		select {
		case event, ok := <-watcher.Events:
			klog.Infof("Watched an event: %v", event)
			if !ok {
				return
			}
			if event.Op&fsnotify.Remove == fsnotify.Remove {
				err = watcher.Add(c.ConfigSetFile)
				if err != nil {
					klog.ErrorS(err, "Failed to watch.", "file", c.ConfigSetFile)
					continue
				}
				klog.Infof("ConfigSet file %s removed. Reload it.", event.Name)
				if err = c.loadConfigSetFile(); err != nil {
					klog.ErrorS(err, "Failed to load config set file.")
				}
			} else if event.Op&fsnotify.Write == fsnotify.Write {
				klog.Infof("ConfigSet file %s modified. Reload it.", event.Name)
				if err = c.loadConfigSetFile(); err != nil {
					klog.ErrorS(err, "Failed to load config set file.")
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			klog.Error(err)
		}
	}
}

func (c *Controller) loadConfigSetFile() error {
	newConfigSet, err := recommend.LoadConfigSetFromFile(c.ConfigSetFile)
	if err != nil {
		klog.ErrorS(err, "Failed to load recommendation config file", "file", c.ConfigSetFile)
		return err
	}
	c.configSet = newConfigSet
	klog.Info("ConfigSet updated.")
	return nil
}
*/
