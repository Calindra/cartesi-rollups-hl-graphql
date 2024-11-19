package model

import (
	cModel "github.com/calindra/cartesi-rollups-hl-graphql/pkg/convenience/model"
)

type Sequencer interface {
	FinishAndGetNext(accept bool) (cModel.Input, error)
}

type InputBoxSequencer struct {
	model *NonodoModel
}

func NewInputBoxSequencer(model *NonodoModel) *InputBoxSequencer {
	return &InputBoxSequencer{model: model}
}

func NewEspressoSequencer(model *NonodoModel) *EspressoSequencer {
	return &EspressoSequencer{model: model}
}

func (ibs *InputBoxSequencer) FinishAndGetNext(accept bool) (cModel.Input, error) {
	return FinishAndGetNext(ibs.model, accept)
}

func (es *EspressoSequencer) FinishAndGetNext(accept bool) (cModel.Input, error) {
	return FinishAndGetNext(es.model, accept)
}

type EspressoSequencer struct {
	model *NonodoModel
}

func FinishAndGetNext(m *NonodoModel, accept bool) (cModel.Input, error) {
	panic("remove this method please")
}
