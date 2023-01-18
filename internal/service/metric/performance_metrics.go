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
	"github.com/dell/csm-metrics-powermax/internal/k8s"
	"github.com/dell/csm-metrics-powermax/internal/service/types"
	"go.opentelemetry.io/otel/attribute"
	"strings"
	"sync"
	"time"
)

// PerformanceMetrics performance metrics
type PerformanceMetrics struct {
	*BaseMetrics
}

// performanceMetricsInstance single instance of Performance metrics
var performanceMetricsInstance *PerformanceMetrics

// CreatePerformanceMetricsInstance return a singleton instance of PerformanceMetrics.
func CreatePerformanceMetricsInstance(service types.Service) *PerformanceMetrics {
	if performanceMetricsInstance == nil {
		lock.Lock()
		defer lock.Unlock()

		if performanceMetricsInstance == nil {
			base := NewBaseMetrics(service)
			performanceMetricsInstance = &PerformanceMetrics{base}
			base.Collector = performanceMetricsInstance
		}
	}

	return performanceMetricsInstance
}

func (m *PerformanceMetrics) Collect(ctx context.Context) error {
	pvs, err := m.VolumeFinder.GetPersistentVolumes(ctx)
	if err != nil {
		m.Logger.WithError(err).Error("find no PVs, will do nothing")
		return err
	}

	for range m.pushPerformanceMetrics(ctx, m.gatherPerformanceMetrics(ctx, pvs)) {
		// consume the channel until it is empty and closed
	}
	return nil
}

func (m *PerformanceMetrics) gatherPerformanceMetrics(ctx context.Context, pvs []k8s.VolumeInfo) <-chan *types.VolumePerfMetricsRecord {
	start := time.Now()
	defer m.TimeSince(start, "gatherPerformanceMetrics")

	ch := make(chan *types.VolumePerfMetricsRecord)
	var wg sync.WaitGroup
	sem := make(chan struct{}, m.MaxPowerMaxConnections)

	// Store the last available time for query
	array2LastAvailTime := make(map[string]int64)
	// Cache the list of storage groups, map storage group works as a set
	array2Sgs := make(map[string]map[string]struct{})
	// Cache volume for quickly updating
	id2Volume := make(map[string]*k8s.VolumeInfo)
	for _, volume := range pvs {
		volumeProperties := strings.Split(volume.VolumeHandle, "-")
		if len(volumeProperties) < 2 {
			m.Logger.WithField("volume_handle", volume.VolumeHandle).Warn("unable to get Volume ID and Array ID from volume handle")
			continue
		}
		volumeID := volumeProperties[len(volumeProperties)-1]
		arrayID := volumeProperties[len(volumeProperties)-2]
		if _, ok := array2LastAvailTime[arrayID]; !ok {
			pmaxClient, err := m.GetPowerMaxClient(arrayID)
			if err != nil {
				m.Logger.WithError(err).WithField("arrayID", arrayID).Warn("no client found for PowerMax")
				continue
			}
			timeResult, err := pmaxClient.GetArrayPerfKeys(ctx)
			if err != nil {
				m.Logger.WithError(err).WithField("arrayID", arrayID).Warn("cannot query last available time")
				continue
			}
			// Store the query time for arrays
			for _, arrayInfo := range timeResult.ArrayInfos {
				array2LastAvailTime[arrayInfo.SymmetrixID] = arrayInfo.LastAvailableDate
			}
		}

		// cache the target of query
		if sgs, ok := array2Sgs[arrayID]; ok {
			if _, ok := sgs[volume.StorageGroup]; !ok {
				sgs[volume.StorageGroup] = struct{}{}
			}
		} else {
			array2Sgs[arrayID] = map[string]struct{}{
				volume.StorageGroup: {},
			}
		}

		// Use volumeID, arrayID to quickly find the unique volume
		id2Volume[volumeID+"="+arrayID] = &volume
	}

	go func() {
		for arrayID, sgs := range array2Sgs {
			queryString := ""
			for sgID := range sgs {
				queryString = queryString + sgID + ","
			}
			pmaxClient, err := m.GetPowerMaxClient(arrayID)
			if err != nil {
				m.Logger.WithError(err).WithField("arrayID", arrayID).Warn("no client found for PowerMax")
				continue
			}
			volumesMetrics, err := pmaxClient.GetVolumesMetrics(ctx, arrayID, queryString, []string{"MBRead", "MBWritten",
				"ReadResponseTime", "WriteResponseTime", "Reads", "Writes"},
				array2LastAvailTime[arrayID], array2LastAvailTime[arrayID])
			if err != nil {
				m.Logger.WithError(err).WithField("arrayID", arrayID).Warn("failed to get volume metrics")
				continue
			}
			for _, volumeResult := range volumesMetrics.ResultList.Result {
				volume := id2Volume[volumeResult.VolumeID+"="+arrayID]
				if volume != nil && strings.Contains(volumeResult.StorageGroups, volume.StorageGroup) {
					if len(volumeResult.VolumeResult) < 1 {
						m.Logger.WithError(err).WithField("volumeID", volumeResult.VolumeID).Warn("volume result contains nothing")
					}
					//go func() {
					//	defer func() {
					//		wg.Done()
					//		<-sem
					//	}()
					//
					//}()
					metric := &types.VolumePerfMetricsRecord{
						ArrayID:                   arrayID,
						VolumeID:                  volumeResult.VolumeID,
						StorageGroupID:            volume.StorageGroup,
						StorageClass:              volume.StorageClass,
						Driver:                    volume.Driver,
						PersistentVolumeName:      volume.PersistentVolume,
						PersistentVolumeClaimName: volume.PersistentVolumeClaim,
						MBRead:                    volumeResult.VolumeResult[0].MBRead,
						MBWritten:                 volumeResult.VolumeResult[0].MBWritten,
						ReadResponseTime:          volumeResult.VolumeResult[0].ReadResponseTime,
						WriteResponseTime:         volumeResult.VolumeResult[0].WriteResponseTime,
						Reads:                     volumeResult.VolumeResult[0].Reads,
						Writes:                    volumeResult.VolumeResult[0].Writes,
					}
					ch <- metric
				}
			}
			//for _, sgs := range volumes {
			//	wg.Add(1)
			//	sem <- struct{}{}
			//	go func(volume k8s.VolumeInfo, storageGroupID string) {
			//		defer func() {
			//			wg.Done()
			//			<-sem
			//		}()
			//
			//		metric := &types.VolumePerfMetricsRecord{}
			//
			//		ch <- metric
			//	}(*volume, storageGroupID)
		}

		//if !exported {
		//	// If no volumes metrics were exported, we need to export an "empty" metric to update the OT Collector
		//	// so that stale entries are removed
		//	ch <- &types.VolumePerfMetricsRecord{
		//		ArrayID:                   "",
		//		VolumeID:                  "",
		//		StorageGroupID:            "",
		//		StorageClass:              "",
		//		PersistentVolumeName:      "",
		//		PersistentVolumeClaimName: "",
		//		Driver:                    "",
		//	}
		//}
		wg.Wait()
		close(ch)
		close(sem)
	}()
	return ch
}

func (m *PerformanceMetrics) pushPerformanceMetrics(ctx context.Context, volumePerfMetrics <-chan *types.VolumePerfMetricsRecord) <-chan string {
	start := time.Now()
	defer m.TimeSince(start, "pushPerformanceMetrics")
	var wg sync.WaitGroup

	ch := make(chan string)
	go func() {

		// for volume metrics
		for metric := range volumePerfMetrics {
			wg.Add(1)
			go func(metric types.VolumePerfMetricsRecord) {
				defer wg.Done()

				err := m.MetricsRecorder.RecordNumericMetrics(ctx, collectVolPerfMetrics("powermax_volume", metric))
				m.Logger.Debugf("class performance metrics metrics %+v", metric)

				if err != nil {
					m.Logger.WithError(err).WithField("array_id", metric.ArrayID).Error("recording performance statistics for array")
				} else {
					ch <- fmt.Sprintf(metric.ArrayID)
				}
			}(*metric)
		}
		wg.Wait()
		close(ch)
	}()

	return ch
}

func collectVolPerfMetrics(prefix string, metric types.VolumePerfMetricsRecord) []types.NumericMetric {
	labels := []attribute.KeyValue{
		attribute.String("ArrayID", metric.ArrayID),
		attribute.String("Driver", metric.Driver),
		attribute.String("PlotWithMean", "No"),
	}
	var list []types.NumericMetric

	list = append(list, types.NumericMetric{Labels: labels, Name: prefix + "_read_bandwidth", Value: metric.MBRead})
	list = append(list, types.NumericMetric{Labels: labels, Name: prefix + "_write_bandwidth", Value: metric.MBWritten})
	list = append(list, types.NumericMetric{Labels: labels, Name: prefix + "_read_response_time", Value: metric.ReadResponseTime})
	list = append(list, types.NumericMetric{Labels: labels, Name: prefix + "_write_response_time", Value: metric.WriteResponseTime})
	list = append(list, types.NumericMetric{Labels: labels, Name: prefix + "_read_operations", Value: metric.Reads})
	list = append(list, types.NumericMetric{Labels: labels, Name: prefix + "_write_operations", Value: metric.Writes})

	return list
}
