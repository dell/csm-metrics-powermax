/*
 Copyright (c) 2025 Dell Inc. or its subsidiaries. All Rights Reserved.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package metric

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dell/csm-metrics-powermax/internal/k8s"
	"github.com/dell/csm-metrics-powermax/internal/service/metrictypes"
	"github.com/sirupsen/logrus"
)

const (
	// ExpectedVolumeHandleProperties is the number of properties that the VolumeHandle contains
	ExpectedVolumeHandleProperties = 4
)

// TopologyMetrics topology metrics
type TopologyMetrics struct {
	*BaseMetrics
}

// topologyMetricsInstance single instance of Performance metrics
var topologyMetricsInstance *TopologyMetrics

// CreateTopologyMetricsInstance return a singleton instance of TopologyMetrics.
func CreateTopologyMetricsInstance(service metrictypes.Service) *TopologyMetrics {
	if topologyMetricsInstance == nil {
		lock.Lock()
		defer lock.Unlock()

		if topologyMetricsInstance == nil {
			base := NewBaseMetrics(service)
			topologyMetricsInstance = &TopologyMetrics{base}
			base.Collector = topologyMetricsInstance
		}
	}

	return topologyMetricsInstance
}

// Collect metric collection and processing
func (m *TopologyMetrics) Collect(ctx context.Context) error {
	pvs, err := m.VolumeFinder.GetPersistentVolumes(ctx)
	if err != nil {
		m.Logger.WithError(err).Error("find no PVs, will do nothing")
		return err
	}

	for range m.pushTopologyMetrics(ctx, m.gatherTopologyMetrics(m.volumeServer(ctx, pvs))) {
		// consume the channel until it is empty and closed
	} // revive:disable-line:empty-block
	return nil
}

// pushTopologyMetrics will push the provided channel of volume metrics to a data collector
func (m *TopologyMetrics) pushTopologyMetrics(ctx context.Context, topologyMetrics <-chan *metrictypes.TopologyMetricsRecord) <-chan *metrictypes.TopologyMetricsRecord {
	start := time.Now()
	defer m.timeSince(start, "pushTopologyMetrics")
	var wg sync.WaitGroup

	ch := make(chan *metrictypes.TopologyMetricsRecord)
	go func() {
		for metrics := range topologyMetrics {
			wg.Add(1)
			go func(metrics *metrictypes.TopologyMetricsRecord) {
				defer wg.Done()
				err := m.MetricsRecorder.RecordTopologyMetrics(ctx, metrics.TopologyMeta, metrics)
				if err != nil {
					m.Logger.WithError(err).WithField("volume_id", metrics.TopologyMeta.PersistentVolume).Error("recording topology metrics for volume")
				} else {
					ch <- metrics
				}
			}(metrics)
		}
		wg.Wait()
		close(ch)
	}()

	return ch
}

// gatherTopologyMetrics will return a channel of topology metrics
func (s *TopologyMetrics) gatherTopologyMetrics(volumes <-chan k8s.VolumeInfo) <-chan *metrictypes.TopologyMetricsRecord {
	start := time.Now()
	defer s.timeSince(start, "gatherTopologyMetrics")

	ch := make(chan *metrictypes.TopologyMetricsRecord)
	var wg sync.WaitGroup

	go func() {
		for volume := range volumes {
			wg.Add(1)
			go func(volume k8s.VolumeInfo) {
				defer wg.Done()

				topologyMeta := &metrictypes.TopologyMeta{
					Namespace:               volume.Namespace,
					PersistentVolumeClaim:   volume.VolumeClaimName,
					VolumeClaimName:         volume.PersistentVolume,
					PersistentVolumeStatus:  volume.PersistentVolumeStatus,
					PersistentVolume:        volume.PersistentVolume,
					StorageClass:            volume.StorageClass,
					Driver:                  volume.Driver,
					ProvisionedSize:         volume.ProvisionedSize,
					StorageSystemVolumeName: volume.StorageSystemVolumeName,
					StoragePoolName:         volume.SRP,
					StorageSystem:           volume.SymID,
					Protocol:                volume.Protocol,
					CreatedTime:             volume.CreatedTime,
				}

				pvAvailable := int64(1)

				metric := &metrictypes.TopologyMetricsRecord{
					TopologyMeta: topologyMeta,
					PVAvailable:  pvAvailable,
				}

				ch <- metric
			}(volume)
		}

		wg.Wait()
		close(ch)
	}()
	return ch
}

// volumeServer will return a channel of volumes that can provide statistics about each volume
func (s *TopologyMetrics) volumeServer(_ context.Context, volumes []k8s.VolumeInfo) <-chan k8s.VolumeInfo {
	volumeChannel := make(chan k8s.VolumeInfo, len(volumes))
	go func() {
		for _, volume := range volumes {
			volumeChannel <- volume
		}
		close(volumeChannel)
	}()
	return volumeChannel
}

// timeSince will log the amount of time spent in a given function
func (m *TopologyMetrics) timeSince(start time.Time, fName string) {
	m.Logger.WithFields(logrus.Fields{
		"duration": fmt.Sprintf("%v", time.Since(start)),
		"function": fName,
	}).Info("function duration")
}
