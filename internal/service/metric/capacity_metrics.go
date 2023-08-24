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
	"github.com/dell/csm-metrics-powermax/internal/service/types"
	"go.opentelemetry.io/otel/attribute"
)

// CapacityMetrics CapacityMetrics
type CapacityMetrics struct {
	*BaseMetrics
}

// capacityMetricsInstance single instance of CapacityMetrics
var capacityMetricsInstance *CapacityMetrics

// CreateCapacityMetricsInstance return a singleton instance of CapacityMetrics.
func CreateCapacityMetricsInstance(service types.Service) *CapacityMetrics {
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

func (m *CapacityMetrics) gatherCapacityMetrics(ctx context.Context, pvs []k8s.VolumeInfo) <-chan *types.VolumeCapacityMetricsRecord {
	start := time.Now()
	defer m.TimeSince(start, "gatherCapacityMetrics")

	ch := make(chan *types.VolumeCapacityMetricsRecord)
	var wg sync.WaitGroup
	sem := make(chan struct{}, m.MaxPowerMaxConnections)

	go func() {
		exported := false

		for _, volume := range pvs {
			exported = true
			wg.Add(1)
			sem <- struct{}{}
			go func(volume k8s.VolumeInfo) {
				defer func() {
					wg.Done()
					<-sem
				}()

				// VolumeHandle is of the format "volumeIdentifier-serial-volumeId"
				// csi-BYM-yiming-398993ad1b-powermaxtest-000197902573-00822
				volumeProperties := strings.Split(volume.VolumeHandle, "-")
				if len(volumeProperties) < 2 {
					m.Logger.WithField("volume_handle", volume.VolumeHandle).Warn("unable to get Volume ID and Array ID from volume handle")
					return
				}

				volumeID := volumeProperties[len(volumeProperties)-1]
				arrayID := volumeProperties[len(volumeProperties)-2]

				pmaxClient, err := m.GetPowerMaxClient(arrayID)
				if err != nil {
					m.Logger.WithError(err).WithField("arrayID", arrayID).Warn("no client found for PowerMax")
					return
				}

				vol, err := pmaxClient.GetVolumeByID(ctx, arrayID, volumeID)
				if err != nil {
					m.Logger.WithError(err).WithField("arrayID", arrayID).WithField("volumeID", volumeID).Error("getting capacity metrics for volume")
					return
				}

				metric := &types.VolumeCapacityMetricsRecord{
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
				}

				ch <- metric
			}(volume)
		}

		if !exported {
			// If no volumes metrics were exported, we need to export an "empty" metric to update the OT Collector
			// so that stale entries are removed
			ch <- &types.VolumeCapacityMetricsRecord{
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

func (m *CapacityMetrics) pushCapacityMetrics(ctx context.Context, volumeCapacityMetrics <-chan *types.VolumeCapacityMetricsRecord) <-chan string {
	start := time.Now()
	defer m.TimeSince(start, "pushCapacityMetrics")
	var wg sync.WaitGroup

	ch := make(chan string)
	go func() {
		// Sum based on array id for total capacity metrics for array and storage class
		arrayIDMap := make(map[string]types.VolumeCapacityMetricsRecord)
		storageClassMap := make(map[string]types.VolumeCapacityMetricsRecord)
		storageGroupMap := make(map[string]types.VolumeCapacityMetricsRecord)
		srpMap := make(map[string]types.VolumeCapacityMetricsRecord)
		volumeIDMap := make(map[string]types.VolumeCapacityMetricsRecord)

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
			go func(metric types.VolumeCapacityMetricsRecord) {
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
				err := m.MetricsRecorder.RecordNumericMetrics(ctx, collectMetrics("powermax_volume_", labels, metric))
				m.Logger.Debugf("volume capacity metrics %+v", metric)

				if err != nil {
					m.Logger.WithError(err).WithField("array_id", metric.ArrayID).WithField("volume_id", metric.VolumeID).Error("recording capacity metrics for volume")
				} else {
					ch <- fmt.Sprintf(metric.VolumeID)
				}
			}(metric)
		}

		// for storage group
		for _, metric := range storageGroupMap {
			wg.Add(1)
			go func(metric types.VolumeCapacityMetricsRecord) {
				defer wg.Done()

				labels := []attribute.KeyValue{
					attribute.String("ArrayID", metric.ArrayID),
					attribute.String("Driver", metric.Driver),
					attribute.String("StorageGroupID", metric.StorageGroupID),
					attribute.String("SrpID", metric.SrpID),
					attribute.String("PlotWithMean", "No"),
				}
				err := m.MetricsRecorder.RecordNumericMetrics(ctx, collectMetrics("powermax_storage_group_", labels, metric))
				m.Logger.Debugf("storage group capacity metrics %+v", metric)

				if err != nil {
					m.Logger.WithError(err).WithField("array_id", metric.ArrayID).WithField("storage_group_id", metric.StorageGroupID).Error("recording capacity statistics for storage group")
				} else {
					ch <- fmt.Sprintf(metric.StorageGroupID)
				}
			}(metric)
		}

		// for srp
		for _, metric := range srpMap {
			wg.Add(1)
			go func(metric types.VolumeCapacityMetricsRecord) {
				defer wg.Done()

				labels := []attribute.KeyValue{
					attribute.String("ArrayID", metric.ArrayID),
					attribute.String("Driver", metric.Driver),
					attribute.String("SrpID", metric.SrpID),
					attribute.String("PlotWithMean", "No"),
				}
				err := m.MetricsRecorder.RecordNumericMetrics(ctx, collectMetrics("powermax_srp_", labels, metric))
				m.Logger.Debugf("srp capacity metrics %+v", metric)

				if err != nil {
					m.Logger.WithError(err).WithField("array_id", metric.ArrayID).WithField("srp_id", metric.SrpID).Error("recording capacity statistics for srp")
				} else {
					ch <- fmt.Sprintf(metric.SrpID)
				}
			}(metric)
		}

		// for array id
		for _, metric := range arrayIDMap {
			wg.Add(1)
			go func(metric types.VolumeCapacityMetricsRecord) {
				defer wg.Done()

				labels := []attribute.KeyValue{
					attribute.String("ArrayID", metric.ArrayID),
					attribute.String("Driver", metric.Driver),
					attribute.String("PlotWithMean", "No"),
				}
				err := m.MetricsRecorder.RecordNumericMetrics(ctx, collectMetrics("powermax_array_", labels, metric))
				m.Logger.Debugf("array capacity metrics %+v", metric)

				if err != nil {
					m.Logger.WithError(err).WithField("array_id", metric.ArrayID).Error("recording capacity statistics for array")
				} else {
					ch <- fmt.Sprintf(metric.ArrayID)
				}
			}(metric)
		}

		// for storage class
		for _, metric := range storageClassMap {
			wg.Add(1)
			go func(metric types.VolumeCapacityMetricsRecord) {
				defer wg.Done()

				labels := []attribute.KeyValue{
					attribute.String("ArrayID", metric.ArrayID),
					attribute.String("Driver", metric.Driver),
					attribute.String("StorageClass", metric.StorageClass),
					attribute.String("PlotWithMean", "No"),
				}
				err := m.MetricsRecorder.RecordNumericMetrics(ctx, collectMetrics("powermax_storage_class_", labels, metric))
				m.Logger.Debugf("storage class capacity metrics metrics %+v", metric)

				if err != nil {
					m.Logger.WithError(err).WithField("array_id", metric.ArrayID).Error("recording capacity statistics for storage class")
				} else {
					ch <- fmt.Sprintf(metric.ArrayID)
				}
			}(metric)
		}

		wg.Wait()
		close(ch)
	}()

	return ch
}

func cumulate(key string, cacheMap map[string]types.VolumeCapacityMetricsRecord, metric *types.VolumeCapacityMetricsRecord) {
	if volMetrics, ok := cacheMap[key]; !ok {
		cacheMap[key] = *metric
	} else {
		volMetrics.Total = volMetrics.Total + metric.Total
		volMetrics.Used = volMetrics.Used + metric.Used
		cacheMap[key] = volMetrics
	}
}

func collectMetrics(prefix string, labels []attribute.KeyValue, metric types.VolumeCapacityMetricsRecord) []types.NumericMetric {
	var list []types.NumericMetric

	list = append(list, types.NumericMetric{Labels: labels, Name: prefix + "total_capacity_gigabytes", Value: metric.Total})
	list = append(list, types.NumericMetric{Labels: labels, Name: prefix + "used_capacity_gigabytes", Value: metric.Used})

	if metric.Total > 0 {
		list = append(list, types.NumericMetric{Labels: labels, Name: prefix + "used_capacity_percentage", Value: metric.Used * 100 / metric.Total})
	}

	return list
}
