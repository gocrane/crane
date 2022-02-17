package metricprovider

import (
	"context"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

type CronTrigger struct {
	Name     string
	Location *time.Location
	Start    string
	End      string
}

// IsActive return true if the now is in cron trigger schedule start and end, false if else
func (c *CronTrigger) IsActive(ctx context.Context, now time.Time) (bool, error) {
	nowWithLocation := now.In(c.Location)

	schedStart, err := cron.ParseStandard(c.Start)
	if err != nil {
		return false, fmt.Errorf("cron %v unparseable schedule: %v. err: %v", c.Name, c.Start, err)
	}
	schedEnd, err := cron.ParseStandard(c.End)
	if err != nil {
		return false, fmt.Errorf("cron %v unparseable schedule: %v. err: %v", c.Name, c.End, err)
	}
	nextStart := schedStart.Next(nowWithLocation)
	nextEnd := schedEnd.Next(nowWithLocation)

	nowTimestamp := nowWithLocation.Unix()
	switch {
	case nextStart.Unix() < nextEnd.Unix() && nowTimestamp < nextStart.Unix():
		return false, nil
	case nowTimestamp <= nextEnd.Unix():
		return true, nil
	default:
		return false, nil
	}
}
