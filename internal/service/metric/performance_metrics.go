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
	"strings"
	"sync"
	"time"

	"github.com/dell/csm-metrics-powermax/internal/k8s"
	"github.com/dell/csm-metrics-powermax/internal/service/metrictypes"
)

// PerformanceMetrics performance metrics
type PerformanceMetrics struct {
	*BaseMetrics
}

// performanceMetricsInstance single instance of Performance metrics
var performanceMetricsInstance *PerformanceMetrics

// CreatePerformanceMetricsInstance return a singleton instance of PerformanceMetrics.
func CreatePerformanceMetricsInstance(service metrictypes.Service) *PerformanceMetrics {
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

// Collect performance metric collection and processing
func (m *PerformanceMetrics) Collect(ctx context.Context) error {
	pvs, err := m.VolumeFinder.GetPersistentVolumes(ctx)
	if err != nil {
		m.Logger.WithError(err).Error("find no PVs, will do nothing")
		return err
	}

	// locally reorganize the data
	// Cache the list of storage groups, map storage group works as a set
	array2Sgs := make(map[string]map[string]struct{})
	// Cache volume for quickly updating
	id2Volume := make(map[string]*k8s.VolumeInfo)
	for index, volume := range pvs {
		volumeProperties := strings.Split(volume.VolumeHandle, "-")
		if len(volumeProperties) < 2 {
			m.Logger.WithField("volume_handle", volume.VolumeHandle).Warn("unable to get Volume ID and Array ID from volume handle")
			continue
		}
		volumeID := volumeProperties[len(volumeProperties)-1]
		arrayID := volumeProperties[len(volumeProperties)-2]

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

		// Use volumeID, arrayID to quickly locate the unique volume
		id2Volume[volumeID+"="+arrayID] = &pvs[index]
	}
	var wg sync.WaitGroup
	wg.Add(2)

	// Collect volume performance metric
	go func() {
		defer wg.Done()
		for range m.pushVolumePerformanceMetrics(ctx, m.gatherVolumePerformanceMetrics(ctx, array2Sgs, id2Volume)) {
			// consume the channel until it is empty and closed
		} // revive:disable-line:empty-block
	}()

	// Collect storage group performance metric
	go func() {
		defer wg.Done()
		for range m.pushStorageGroupPerformanceMetrics(ctx, m.gatherStorageGroupPerformanceMetrics(ctx, array2Sgs)) {
			// consume the channel until it is empty and closed
		} // revive:disable-line:empty-block
	}()
	wg.Wait()
	return nil
}

func (m *PerformanceMetrics) gatherVolumePerformanceMetrics(ctx context.Context, array2Sgs map[string]map[string]struct{}, id2Volume map[string]*k8s.VolumeInfo) <-chan *metrictypes.VolumePerfMetricsRecord {
	start := time.Now()
	defer m.TimeSince(start, "gatherVolumePerformanceMetrics")

	ch := make(chan *metrictypes.VolumePerfMetricsRecord)
	var wg sync.WaitGroup
	sem := make(chan struct{}, m.MaxPowerMaxConnections)

	// Store the last available time for query
	array2LastAvailTime := make(map[string]int64)
	for arrayID := range array2Sgs {
		if _, ok := array2LastAvailTime[arrayID]; ok {
			// already stored, skip
			continue
		}
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
		// Store the last available time for arrays
		for _, arrayInfo := range timeResult.ArrayInfos {
			array2LastAvailTime[arrayInfo.SymmetrixID] = arrayInfo.LastAvailableDate
			m.Logger.Debugf("last available time of the array %s is %d", arrayInfo.SymmetrixID, arrayInfo.LastAvailableDate)
		}
	}

	go func() {
		exported := false
		for arrayID, storageGroups := range array2Sgs {
			exported = true
			wg.Add(1)
			sem <- struct{}{}
			go func(arrayID string, sgs map[string]struct{}) {
				defer func() {
					wg.Done()
					<-sem
				}()
				if _, ok := array2LastAvailTime[arrayID]; !ok {
					m.Logger.WithField("arrayID", arrayID).Warn("last available time for array is not found")
					return
				}
				pmaxClient, err := m.GetPowerMaxClient(arrayID)
				if err != nil {
					m.Logger.WithError(err).WithField("arrayID", arrayID).Warn("no client found for PowerMax")
					return
				}
				querySgParams := ""
				for sgID := range sgs {
					querySgParams = querySgParams + sgID + ","
				}
				volumesMetrics, err := pmaxClient.GetVolumesMetrics(ctx, arrayID, querySgParams, []string{
					"MBRead", "MBWritten",
					"ReadResponseTime", "WriteResponseTime", "Reads", "Writes",
				},
					array2LastAvailTime[arrayID], array2LastAvailTime[arrayID])
				if err != nil {
					m.Logger.WithError(err).WithField("arrayID", arrayID).Warn("failed to get volume metrics")
					return
				}
				for _, volumeResult := range volumesMetrics.ResultList.Result {
					volume := id2Volume[volumeResult.VolumeID+"="+arrayID]
					if volume != nil && strings.Contains(volumeResult.StorageGroups, volume.StorageGroup) {
						if len(volumeResult.VolumeResult) < 1 {
							m.Logger.WithError(err).WithField("volumeID", volumeResult.VolumeID).Warn("volume result contains nothing")
							continue
						}
						metric := &metrictypes.VolumePerfMetricsRecord{
							ArrayID:                   arrayID,
							VolumeID:                  volumeResult.VolumeID,
							StorageGroupID:            volume.StorageGroup,
							StorageClass:              volume.StorageClass,
							Driver:                    volume.Driver,
							PersistentVolumeName:      volume.PersistentVolume,
							PersistentVolumeClaimName: volume.VolumeClaimName,
							Namespace:                 volume.Namespace,
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
			}(arrayID, storageGroups)
		}

		if !exported {
			// If no volume metrics were exported, we need to export an "empty" metric to update the OT Collector
			// so that stale entries are removed
			ch <- &metrictypes.VolumePerfMetricsRecord{
				ArrayID:                   "",
				VolumeID:                  "",
				StorageGroupID:            "",
				StorageClass:              "",
				Driver:                    "",
				PersistentVolumeName:      "",
				PersistentVolumeClaimName: "",
				Namespace:                 "",
				MBRead:                    0,
				MBWritten:                 0,
				ReadResponseTime:          0,
				WriteResponseTime:         0,
				Reads:                     0,
				Writes:                    0,
			}
		}
		wg.Wait()
		close(ch)
		close(sem)
	}()
	return ch
}

func (m *PerformanceMetrics) pushVolumePerformanceMetrics(_ context.Context, volumePerfMetrics <-chan *metrictypes.VolumePerfMetricsRecord) <-chan string {
	start := time.Now()
	defer m.TimeSince(start, "pushVolumePerformanceMetrics")
	var wg sync.WaitGroup

	ch := make(chan string)
	go func() {
		// for volume metrics
		for metric := range volumePerfMetrics {
			wg.Add(1)
			go func(metric metrictypes.VolumePerfMetricsRecord) {
				defer wg.Done()

				err := m.MetricsRecorder.RecordVolPerfMetrics("powermax_volume", metric)
				m.Logger.Debugf("class volume performance metrics %+v", metric)

				if err != nil {
					m.Logger.WithError(err).WithField("volume_id", metric.VolumeID).Error("recording performance statistics for volume")
				} else {
					ch <- metric.VolumeID
				}
			}(*metric)
		}
		wg.Wait()
		close(ch)
	}()

	return ch
}

func (m *PerformanceMetrics) gatherStorageGroupPerformanceMetrics(ctx context.Context, array2Sgs map[string]map[string]struct{}) <-chan *metrictypes.StorageGroupPerfMetricsRecord {
	start := time.Now()
	defer m.TimeSince(start, "gatherStorageGroupPerformanceMetrics")

	ch := make(chan *metrictypes.StorageGroupPerfMetricsRecord)
	var wg sync.WaitGroup
	sem := make(chan struct{}, m.MaxPowerMaxConnections)

	// Store the last available time for query
	storageGroup2LastAvailTime := make(map[string]int64)
	for arrayID, sgs := range array2Sgs {
		pmaxClient, err := m.GetPowerMaxClient(arrayID)
		if err != nil {
			m.Logger.WithError(err).WithField("arrayID", arrayID).Warn("no client found for PowerMax")
			continue
		}
		timeResult, err := pmaxClient.GetStorageGroupPerfKeys(ctx, arrayID)
		if err != nil {
			m.Logger.WithError(err).WithField("arrayID", arrayID).Warn("cannot query last available time for storage groups in the array")
			continue
		}
		// Store the query time for storage groups
		for _, storageGroupInfo := range timeResult.StorageGroupInfos {
			if _, ok := sgs[storageGroupInfo.StorageGroupID]; ok {
				// store both arrayID & storage for collision risk
				storageGroup2LastAvailTime[arrayID+"="+storageGroupInfo.StorageGroupID] = storageGroupInfo.LastAvailableDate
				m.Logger.Debugf("last available time of the storage group %s is %d", storageGroupInfo.StorageGroupID, storageGroupInfo.LastAvailableDate)
			}
		}
	}

	go func() {
		exported := false
		for arrayID, sgs := range array2Sgs {
			exported = true
			wg.Add(1)
			sem <- struct{}{}
			go func(arrayID string, sgs map[string]struct{}) {
				defer func() {
					wg.Done()
					<-sem
				}()
				pmaxClient, err := m.GetPowerMaxClient(arrayID)
				if err != nil {
					m.Logger.WithError(err).WithField("arrayID", arrayID).Warn("no client found for PowerMax")
					return
				}
				for storageGroupID := range sgs {
					if _, ok := storageGroup2LastAvailTime[arrayID+"="+storageGroupID]; !ok {
						m.Logger.WithField("storageGroupID", storageGroupID).Warn("last available time for storage group is not found")
						continue
					}
					sgMetrics, err := pmaxClient.GetStorageGroupMetrics(ctx, arrayID, storageGroupID, []string{
						"HostMBReads", "HostMBWritten",
						"ReadResponseTime", "WriteResponseTime", "HostReads", "HostWrites", "AvgIOSize",
					},
						storageGroup2LastAvailTime[arrayID+"="+storageGroupID], storageGroup2LastAvailTime[arrayID+"="+storageGroupID])
					if err != nil {
						m.Logger.WithError(err).WithField("storageGroupID ID", storageGroupID).Warn("failed to get storage group metrics")
						continue
					}
					for _, sgResult := range sgMetrics.ResultList.Result {
						metric := &metrictypes.StorageGroupPerfMetricsRecord{
							ArrayID:           arrayID,
							StorageGroupID:    storageGroupID,
							HostMBReads:       sgResult.HostMBReads,
							HostMBWritten:     sgResult.HostMBWritten,
							ReadResponseTime:  sgResult.ReadResponseTime,
							WriteResponseTime: sgResult.WriteResponseTime,
							HostReads:         sgResult.HostReads,
							HostWrites:        sgResult.HostWrites,
							AvgIOSize:         sgResult.AvgIOSize,
						}
						ch <- metric
					}
				}
			}(arrayID, sgs)
		}
		if !exported {
			// If no storage group metrics were exported, we need to export an "empty" metric to update the OT Collector
			// so that stale entries are removed
			ch <- &metrictypes.StorageGroupPerfMetricsRecord{
				ArrayID:           "",
				StorageGroupID:    "",
				HostMBReads:       0,
				HostMBWritten:     0,
				ReadResponseTime:  0,
				WriteResponseTime: 0,
				HostReads:         0,
				HostWrites:        0,
				AvgIOSize:         0,
			}
		}
		wg.Wait()
		close(ch)
		close(sem)
	}()
	return ch
}

func (m *PerformanceMetrics) pushStorageGroupPerformanceMetrics(_ context.Context, storageGroupPerfMetrics <-chan *metrictypes.StorageGroupPerfMetricsRecord) <-chan string {
	start := time.Now()
	defer m.TimeSince(start, "pushStorageGroupPerformanceMetrics")
	var wg sync.WaitGroup

	ch := make(chan string)
	go func() {
		// for storage group metrics
		for metric := range storageGroupPerfMetrics {
			wg.Add(1)
			go func(metric metrictypes.StorageGroupPerfMetricsRecord) {
				defer wg.Done()

				err := m.MetricsRecorder.RecordStorageGroupPerfMetrics("powermax_storage_group", metric)
				m.Logger.Debugf("storage group performance metrics metrics %+v", metric)

				if err != nil {
					m.Logger.WithError(err).WithField("storage_group_id", metric.ArrayID).Error("recording performance statistics for storage group")
				} else {
					ch <- metric.StorageGroupID
				}
			}(*metric)
		}
		wg.Wait()
		close(ch)
	}()

	return ch
}
