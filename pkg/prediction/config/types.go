package config

import (
	"time"

	"github.com/gocrane/api/prediction/v1alpha1"
)

type AlgorithmModelConfig struct {
	UpdateInterval time.Duration
}

type ModelInitMode string

const (
	// means recover or init the algorithm model directly from history datasource, this process may block because it is time consuming for data fetching & model gen
	ModelInitModeHistory ModelInitMode = "history"
	// means recover or init the algorithm model from real time datasource async, predictor can not do predicting before the data is accumulating to window length
	// this is more safe to do some data accumulating and make the prediction data is robust.
	ModelInitModeLazyTraining ModelInitMode = "lazytraining"
	// means recover or init the model from a checkpoint, it can be restored directly and immediately to do predict.
	ModelInitModeCheckpoint ModelInitMode = "checkpoint"
)

type Config struct {
	InitMode   *ModelInitMode
	DSP        *v1alpha1.DSP
	Percentile *v1alpha1.Percentile
}
