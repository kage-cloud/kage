package canaryutil

import (
	"github.com/kage-cloud/kage/xds/model"
	"math"
)

func DeriveReplicaCountFromTraffic(maxReplicas int32, trafficPercentage uint32) int32 {
	percentReps := float32(maxReplicas) * (float32(trafficPercentage) / float32(model.TotalRoutingWeight))
	return int32(math.Ceil(float64(percentReps)))
}
