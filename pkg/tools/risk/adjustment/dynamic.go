package adjustment

import (
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type DynamicAdjustment interface {
	AdjustPosition(common.Position, common.Tick) (fixed.Point, fixed.Point, bool)
}
