package arima

import (
	"errors"

	"github.com/peter-kozarec/equinox/pkg/utility/circular"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

var (
	ErrModelNotEstimated = errors.New("not enough data points")
)

type Model struct {
	p, d, q   uint
	ptCounter uint
	winSize   uint
	aR        *circular.PointBuffer
	mA        *circular.PointBuffer
	rawData   *circular.PointBuffer
	diffData  *circular.PointBuffer
	residuals *circular.PointBuffer
}

func NewModel(p, d, q, winSize uint) *Model {
	return &Model{
		p:         p,
		d:         d,
		q:         q,
		ptCounter: 0,
		winSize:   winSize,
		aR:        circular.NewPointBuffer(p),
		mA:        circular.NewPointBuffer(q),
		rawData:   circular.NewPointBuffer(winSize),     // Non-stationary data points
		diffData:  circular.NewPointBuffer(winSize - d), // Stationary data points
		residuals: circular.NewPointBuffer(q * 2),
	}
}

func (m *Model) PredictNextPoint() (fixed.Point, error) {
	if !m.aR.B.IsEmpty() || !m.mA.B.IsEmpty() {
		return fixed.Point{}, ErrModelNotEstimated
	}

	mean := m.diffData.Mean()
	forecast := mean

	// AR component
	for i := uint(0); i < m.p; i++ {
		forecast = forecast.Add(m.aR.B.Get(m.aR.B.Size() - i - 1).Mul(m.diffData.B.Get(i).Sub(mean)))
	}

	// MA component
	for i := uint(0); i < m.q; i++ {
		forecast = forecast.Add(m.mA.B.Get(m.mA.B.Size() - i - 1).Mul(m.residuals.B.Get(i)))
	}

	return forecast, nil
}

func (m *Model) AddPoint(p fixed.Point) {
	m.rawData.PushUpdate(p)
	m.ptCounter++

	// Populate stationary series
	if m.rawData.B.Size() > m.d {
		diff := m.rawData.B.Get(0)
		for i := uint(1); i <= m.d; i++ {
			diff = diff.Sub(m.rawData.B.Get(i))
		}
		m.diffData.PushUpdate(diff)
	}

	// Re-estimate the model parameters if necessary
	if m.ptCounter >= m.winSize {
		m.ptCounter = 0 // reset counter
		m.estimate()
	}
}

func (m *Model) estimate() {
	m.estimateArParameters()
	m.estimateMaParameters()
	m.calculateResiduals()
}

func (m *Model) estimateArParameters() {

}

func (m *Model) estimateMaParameters() {

}

func (m *Model) calculateResiduals() {

}
