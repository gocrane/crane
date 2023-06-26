package utils

import (
	"time"

	"github.com/gocrane/crane/pkg/common"
)

func DetectTimestampCompletion(tsList []*common.TimeSeries, historyLength string, timeNow time.Time) (bool, int, error) {
	historyLengthDuration, err := ParseDuration(historyLength)
	if err != nil {
		return false, 0, err
	}
	end := timeNow.Truncate(time.Minute)
	days := int(historyLengthDuration.Hours() / 24)
	if days < 1 {
		days = 1
	}
	daysExistence := make(map[int]bool, days)

	for _, sample := range tsList[0].Samples {
		dayDifference := int((end.Unix() - sample.Timestamp) / (24 * 60 * 60))
		if 0 <= dayDifference && dayDifference <= days {
			daysExistence[dayDifference] = true
		}
	}

	existDays := 0
	for _, exists := range daysExistence {
		if exists {
			existDays++
		}
	}

	return existDays >= (days - 1), existDays, nil // stretch the checking to tolerate some missing data.
}
