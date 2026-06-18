/*
 Copyright (c) 2022-2023 Dell Inc. or its subsidiaries. All Rights Reserved.

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
	"strings"
	"sync"
	"time"

	"github.com/dell/csm-metrics-powermax/internal/k8s"
	"github.com/dell/csm-metrics-powermax/internal/service/metrictypes"
	types "github.com/dell/gopowermax/v2/types/v100"
	"go.opentelemetry.io/otel/attribute"
)

// CapacityMetrics CapacityMetrics
type CapacityMetrics struct {
	*BaseMetrics
}

// capacityMetricsInstance single instance of CapacityMetrics
var capacityMetricsInstance *CapacityMetrics

// CreateCapacityMetricsInstance return a singleton instance of CapacityMetrics.
func CreateCapacityMetricsInstance(service metrictypes.Service) *CapacityMetrics {
	if capacityMetricsInstance == nil {
		lock.Lock()
		defer lock.Unlock()

		if capacityMetricsInstance == nil {
			base := NewBaseMetrics(service)
			capacityMetricsInstance = &CapacityMetrics{base}
			base.Collector = capacityMetricsInstance
		}
	}

	return capacityMetricsInstance
}

// Collect metric collection and processing
func (m *CapacityMetrics) Collect(ctx context.Context) error {
	pvs, err := m.VolumeFinder.GetPersistentVolumes(ctx)
	if err != nil {
		m.Logger.WithError(err).Error("find no PVs, will do nothing")
		return err
	}

	for range m.pushCapacityMetrics(ctx, m.gatherCapacityMetrics(ctx, pvs)) {
		// consume the channel until it is empty and closed
	} // revive:disable-line:empty-block
	return nil
}

func (m *CapacityMetrics) gatherCapacityMetrics(ctx context.Context, pvs []k8s.VolumeInfo) <-chan *metrictypes.VolumeCapacityMetricsRecord {
	start := time.Now()
	defer m.TimeSince(start, "gatherCapacityMetrics")

	ch := make(chan *metrictypes.VolumeCapacityMetricsRecord)
	var wg sync.WaitGroup
	sem := make(chan struct{}, m.MaxPowerMaxConnections)

	// Group the persistent volumes by array so capacity can be retrieved with a single
	// bulk call per array instead of one call per volume.
	// VolumeHandle is of the format "volumeIdentifier-serial-volumeId"
	// csi-BYM-yiming-398993ad1b-powermaxtest-000197902573-00822
	array2Volumes := make(map[string][]k8s.VolumeInfo)
	for _, volume := range pvs {
		volumeProperties := strings.Split(volume.VolumeHandle, "-")
		if len(volumeProperties) < 2 {
			m.Logger.WithField("volume_handle", volume.VolumeHandle).Warn("unable to get Volume ID and Array ID from volume handle")
			continue
		}
		arrayID := volumeProperties[len(volumeProperties)-2]
		array2Volumes[arrayID] = append(array2Volumes[arrayID], volume)
	}

	go func() {
		exported := false

		for arrayID, volumes := range array2Volumes {
			exported = true
			wg.Add(1)
			sem <- struct{}{}
			go func(arrayID string, volumes []k8s.VolumeInfo) {
				defer func() {
					wg.Done()
					<-sem
				}()

				pmaxClient, err := m.GetPowerMaxClient(arrayID)
				if err != nil {
					m.Logger.WithError(err).WithField("arrayID", arrayID).Warn("no client found for PowerMax")
					return
				}

				for _, metric := range m.collectArrayCapacityMetrics(ctx, pmaxClient, arrayID, volumes) {
					ch <- metric
				}
			}(arrayID, volumes)
		}

		if !exported {
			// If no volumes metrics were exported, we need to export an "empty" metric to update the OT Collector
			// so that stale entries are removed
			ch <- &metrictypes.VolumeCapacityMetricsRecord{
				ArrayID:                   "",
				VolumeID:                  "",
				StorageGroupID:            "",
				SrpID:                     "",
				StorageClass:              "",
				PersistentVolumeName:      "",
				PersistentVolumeStatus:    "",
				PersistentVolumeClaimName: "",
				Namespace:                 "",
				Driver:                    "",
				Total:                     0,
				Used:                      0,
				UsedPercent:               0,
			}
		}
		wg.Wait()
		close(ch)
		close(sem)
	}()
	return ch
}

// collectArrayCapacityMetrics retrieves capacity metrics for all the supplied volumes on a single
// array. It issues one bulk call (GetVolumesCapacityBulk) and maps the result back to each volume.
// If the bulk endpoint is unavailable (e.g. older Unisphere), it falls back to per-volume
// GetVolumeByID calls so behavior is preserved on legacy arrays.
func (m *CapacityMetrics) collectArrayCapacityMetrics(ctx context.Context, pmaxClient metrictypes.PowerMaxClient, arrayID string, volumes []k8s.VolumeInfo) []*metrictypes.VolumeCapacityMetricsRecord {
	metrics := make([]*metrictypes.VolumeCapacityMetricsRecord, 0, len(volumes))

	bulk, err := pmaxClient.GetVolumesCapacityBulk(ctx, arrayID)
	if err != nil {
		m.Logger.WithError(err).WithField("arrayID", arrayID).Warn("bulk capacity collection failed, falling back to per-volume collection")
		return m.collectArrayCapacityMetricsLegacy(ctx, pmaxClient, arrayID, volumes)
	}
	if bulk == nil {
		m.Logger.WithField("arrayID", arrayID).Warn("bulk capacity collection returned nil, falling back to per-volume collection")
		return m.collectArrayCapacityMetricsLegacy(ctx, pmaxClient, arrayID, volumes)
	}

	// Index the bulk result by volume ID for O(1) lookup.
	capByVolumeID := make(map[string]types.VolumeEnhanced, len(bulk.Volumes))
	for _, vol := range bulk.Volumes {
		capByVolumeID[vol.ID] = vol
	}

	for _, volume := range volumes {
		volumeProperties := strings.Split(volume.VolumeHandle, "-")
		volumeID := volumeProperties[len(volumeProperties)-1]

		vol, ok := capByVolumeID[volumeID]
		if !ok {
			m.Logger.WithField("arrayID", arrayID).WithField("volumeID", volumeID).Warn("volume not found in bulk capacity response")
			continue
		}

		usedPercent := 0.0
		if vol.CapGB > 0 {
			usedPercent = vol.EffectiveUsedCapacityGB / vol.CapGB * 100
		}
		metrics = append(metrics, &metrictypes.VolumeCapacityMetricsRecord{
			ArrayID:                   arrayID,
			VolumeID:                  volumeID,
			StorageGroupID:            volume.StorageGroup,
			SrpID:                     volume.SRP,
			StorageClass:              volume.StorageClass,
			PersistentVolumeName:      volume.PersistentVolume,
			PersistentVolumeStatus:    volume.PersistentVolumeStatus,
			PersistentVolumeClaimName: volume.VolumeClaimName,
			Namespace:                 volume.Namespace,
			Driver:                    volume.Driver,
			Total:                     vol.CapGB,
			Used:                      vol.EffectiveUsedCapacityGB,
			UsedPercent:               usedPercent,
		})
	}
	return metrics
}

// collectArrayCapacityMetricsLegacy retrieves capacity metrics one volume at a time using
// GetVolumeByID. It is used as a fallback when the bulk endpoint is not available.
func (m *CapacityMetrics) collectArrayCapacityMetricsLegacy(ctx context.Context, pmaxClient metrictypes.PowerMaxClient, arrayID string, volumes []k8s.VolumeInfo) []*metrictypes.VolumeCapacityMetricsRecord {
	metrics := make([]*metrictypes.VolumeCapacityMetricsRecord, 0, len(volumes))
	for _, volume := range volumes {
		volumeProperties := strings.Split(volume.VolumeHandle, "-")
		volumeID := volumeProperties[len(volumeProperties)-1]

		vol, err := pmaxClient.GetVolumeByID(ctx, arrayID, volumeID)
		if err != nil {
			m.Logger.WithError(err).WithField("arrayID", arrayID).WithField("volumeID", volumeID).Error("getting capacity metrics for volume")
			continue
		}
		metrics = append(metrics, &metrictypes.VolumeCapacityMetricsRecord{
			ArrayID:                   arrayID,
			VolumeID:                  volumeID,
			StorageGroupID:            volume.StorageGroup,
			SrpID:                     volume.SRP,
			StorageClass:              volume.StorageClass,
			PersistentVolumeName:      volume.PersistentVolume,
			PersistentVolumeStatus:    volume.PersistentVolumeStatus,
			PersistentVolumeClaimName: volume.VolumeClaimName,
			Namespace:                 volume.Namespace,
			Driver:                    volume.Driver,
			Total:                     vol.CapacityGB,
			Used:                      float64(vol.AllocatedPercent) / 100 * vol.CapacityGB,
			UsedPercent:               float64(vol.AllocatedPercent),
		})
	}
	return metrics
}

func (m *CapacityMetrics) pushCapacityMetrics(_ context.Context, volumeCapacityMetrics <-chan *metrictypes.VolumeCapacityMetricsRecord) <-chan string {
	start := time.Now()
	defer m.TimeSince(start, "pushCapacityMetrics")
	var wg sync.WaitGroup

	ch := make(chan string)
	go func() {
		// Sum based on array id for total capacity metrics for array and storage class
		arrayIDMap := make(map[string]metrictypes.VolumeCapacityMetricsRecord)
		storageClassMap := make(map[string]metrictypes.VolumeCapacityMetricsRecord)
		storageGroupMap := make(map[string]metrictypes.VolumeCapacityMetricsRecord)
		srpMap := make(map[string]metrictypes.VolumeCapacityMetricsRecord)
		volumeIDMap := make(map[string]metrictypes.VolumeCapacityMetricsRecord)

		for metric := range volumeCapacityMetrics {
			// for array id cumulative
			cumulate(metric.ArrayID, arrayIDMap, metric)
			// for Storage Class cumulative
			cumulate(metric.StorageClass, storageClassMap, metric)
			// for Volume cumulative
			cumulate(fmt.Sprintf("%s-%s", metric.ArrayID, metric.VolumeID), volumeIDMap, metric)
			// for StorageGroup cumulative
			cumulate(fmt.Sprintf("%s-%s", metric.ArrayID, metric.StorageGroupID), storageGroupMap, metric)
			// for Srp cumulative
			cumulate(fmt.Sprintf("%s-%s", metric.ArrayID, metric.SrpID), srpMap, metric)
		}

		// for volume
		for _, metric := range volumeIDMap {
			wg.Add(1)
			go func(metric metrictypes.VolumeCapacityMetricsRecord) {
				defer wg.Done()

				labels := []attribute.KeyValue{
					attribute.String("ArrayID", metric.ArrayID),
					attribute.String("SrpID", metric.SrpID),
					attribute.String("StorageGroupID", metric.StorageGroupID),
					attribute.String("VolumeID", metric.VolumeID),
					attribute.String("Driver", metric.Driver),
					attribute.String("StorageClass", metric.StorageClass),
					attribute.String("PersistentVolumeName", metric.PersistentVolumeName),
					attribute.String("PersistentVolumeStatus", metric.PersistentVolumeStatus),
					attribute.String("PersistentVolumeClaimName", metric.PersistentVolumeClaimName),
					attribute.String("Namespace", metric.Namespace),
					attribute.String("PlotWithMean", "No"),
				}
				err := m.MetricsRecorder.RecordNumericMetrics("powermax_volume_", labels, metric)
				m.Logger.Debugf("volume capacity metrics %+v", metric)

				if err != nil {
					m.Logger.WithError(err).WithField("array_id", metric.ArrayID).WithField("volume_id", metric.VolumeID).Error("recording capacity metrics for volume")
				} else {
					ch <- metric.VolumeID
				}
			}(metric)
		}

		// for storage group
		for _, metric := range storageGroupMap {
			wg.Add(1)
			go func(metric metrictypes.VolumeCapacityMetricsRecord) {
				defer wg.Done()

				labels := []attribute.KeyValue{
					attribute.String("ArrayID", metric.ArrayID),
					attribute.String("Driver", metric.Driver),
					attribute.String("StorageGroupID", metric.StorageGroupID),
					attribute.String("SrpID", metric.SrpID),
					attribute.String("PlotWithMean", "No"),
				}
				err := m.MetricsRecorder.RecordNumericMetrics("powermax_storage_group_", labels, metric)
				m.Logger.Debugf("storage group capacity metrics %+v", metric)

				if err != nil {
					m.Logger.WithError(err).WithField("array_id", metric.ArrayID).WithField("storage_group_id", metric.StorageGroupID).Error("recording capacity statistics for storage group")
				} else {
					ch <- metric.StorageGroupID
				}
			}(metric)
		}

		// for srp
		for _, metric := range srpMap {
			wg.Add(1)
			go func(metric metrictypes.VolumeCapacityMetricsRecord) {
				defer wg.Done()

				labels := []attribute.KeyValue{
					attribute.String("ArrayID", metric.ArrayID),
					attribute.String("Driver", metric.Driver),
					attribute.String("SrpID", metric.SrpID),
					attribute.String("PlotWithMean", "No"),
				}
				err := m.MetricsRecorder.RecordNumericMetrics("powermax_srp_", labels, metric)
				m.Logger.Debugf("srp capacity metrics %+v", metric)

				if err != nil {
					m.Logger.WithError(err).WithField("array_id", metric.ArrayID).WithField("srp_id", metric.SrpID).Error("recording capacity statistics for srp")
				} else {
					ch <- metric.SrpID
				}
			}(metric)
		}

		// for array id
		for _, metric := range arrayIDMap {
			wg.Add(1)
			go func(metric metrictypes.VolumeCapacityMetricsRecord) {
				defer wg.Done()

				labels := []attribute.KeyValue{
					attribute.String("ArrayID", metric.ArrayID),
					attribute.String("Driver", metric.Driver),
					attribute.String("PlotWithMean", "No"),
				}
				err := m.MetricsRecorder.RecordNumericMetrics("powermax_array_", labels, metric)
				m.Logger.Debugf("array capacity metrics %+v", metric)

				if err != nil {
					m.Logger.WithError(err).WithField("array_id", metric.ArrayID).Error("recording capacity statistics for array")
				} else {
					ch <- metric.ArrayID
				}
			}(metric)
		}

		// for storage class
		for _, metric := range storageClassMap {
			wg.Add(1)
			go func(metric metrictypes.VolumeCapacityMetricsRecord) {
				defer wg.Done()

				labels := []attribute.KeyValue{
					attribute.String("ArrayID", metric.ArrayID),
					attribute.String("Driver", metric.Driver),
					attribute.String("StorageClass", metric.StorageClass),
					attribute.String("PlotWithMean", "No"),
				}
				err := m.MetricsRecorder.RecordNumericMetrics("powermax_storage_class_", labels, metric)
				m.Logger.Debugf("storage class capacity metrics metrics %+v", metric)

				if err != nil {
					m.Logger.WithError(err).WithField("array_id", metric.ArrayID).Error("recording capacity statistics for storage class")
				} else {
					ch <- metric.ArrayID
				}
			}(metric)
		}

		wg.Wait()
		close(ch)
	}()

	return ch
}

func cumulate(key string, cacheMap map[string]metrictypes.VolumeCapacityMetricsRecord, metric *metrictypes.VolumeCapacityMetricsRecord) {
	if volMetrics, ok := cacheMap[key]; !ok {
		cacheMap[key] = *metric
	} else {
		volMetrics.Total = volMetrics.Total + metric.Total
		volMetrics.Used = volMetrics.Used + metric.Used
		cacheMap[key] = volMetrics
	}
}
