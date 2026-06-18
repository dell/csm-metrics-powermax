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
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dell/csm-metrics-powermax/internal/k8s"
	"github.com/dell/csm-metrics-powermax/internal/service/metrictypes"
)

const sgCountCacheTTL = 5 * time.Minute

// defaultBulkThresholdRatio is the default minimum fraction of monitored-to-total storage
// groups before the bulk GET endpoint is preferred over individual POST calls.
// Bulk fetches ALL SG metrics in one call so it pays off when a large share of
// SGs is monitored; individual calls are cheaper when only a few SGs are
// needed.  0.25 (25 %) was chosen as a reasonable default balancing response
// size against per-call overhead.
const defaultBulkThresholdRatio = 0.25

// bulkThresholdRatio is the active threshold ratio, optionally overridden by
// the POWERMAX_BULK_THRESHOLD_RATIO environment variable.
var bulkThresholdRatio = defaultBulkThresholdRatio

func init() {
	if v := os.Getenv("POWERMAX_BULK_THRESHOLD_RATIO"); v != "" {
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "POWERMAX_BULK_THRESHOLD_RATIO is not a valid number (%q), using default %v\n", v, defaultBulkThresholdRatio)
		} else if parsed < 0 || parsed > 1 {
			fmt.Fprintf(os.Stderr, "POWERMAX_BULK_THRESHOLD_RATIO must be between 0 and 1 (%v), using default %v\n", parsed, defaultBulkThresholdRatio)
		} else {
			bulkThresholdRatio = parsed
		}
	}
}

// sgCountCacheEntry represents a cached storage group count with timestamp
type sgCountCacheEntry struct {
	count     int
	timestamp time.Time
}

// PerformanceMetrics performance metrics
type PerformanceMetrics struct {
	*BaseMetrics
	sgCountCache      map[string]sgCountCacheEntry
	sgCountCacheMutex sync.RWMutex
}

// performanceMetricsInstance single instance of Performance metrics
var performanceMetricsInstance *PerformanceMetrics

// NewPerformanceMetrics creates a new PerformanceMetrics with the given BaseMetrics and
// properly initialized internal state. Use this for constructing test instances.
func NewPerformanceMetrics(base *BaseMetrics) *PerformanceMetrics {
	return &PerformanceMetrics{
		BaseMetrics:  base,
		sgCountCache: make(map[string]sgCountCacheEntry),
	}
}

// CreatePerformanceMetricsInstance return a singleton instance of PerformanceMetrics.
func CreatePerformanceMetricsInstance(service metrictypes.Service) *PerformanceMetrics {
	if performanceMetricsInstance == nil {
		lock.Lock()
		defer lock.Unlock()

		if performanceMetricsInstance == nil {
			base := NewBaseMetrics(service)
			performanceMetricsInstance = &PerformanceMetrics{
				BaseMetrics:  base,
				sgCountCache: make(map[string]sgCountCacheEntry),
			}
			base.Collector = performanceMetricsInstance
		}
	}

	return performanceMetricsInstance
}

// getTotalSGCount returns the total number of storage groups for an array, using cache if available
func (m *PerformanceMetrics) getTotalSGCount(ctx context.Context, pmaxClient metrictypes.PowerMaxClient, arrayID string) (int, error) {
	// Check cache first
	m.sgCountCacheMutex.RLock()
	if m.sgCountCache != nil {
		entry, exists := m.sgCountCache[arrayID]
		m.sgCountCacheMutex.RUnlock()
		if exists && time.Since(entry.timestamp) < sgCountCacheTTL {
			return entry.count, nil
		}
	} else {
		m.sgCountCacheMutex.RUnlock()
	}

	// Cache miss or expired, fetch from API
	sgIDList, err := pmaxClient.GetStorageGroupIDList(ctx, arrayID, "", false)
	if err != nil {
		return 0, err
	}

	totalSGs := len(sgIDList.StorageGroupIDs)

	// Update cache
	m.sgCountCacheMutex.Lock()
	if m.sgCountCache == nil {
		m.sgCountCache = make(map[string]sgCountCacheEntry)
	}
	m.sgCountCache[arrayID] = sgCountCacheEntry{
		count:     totalSGs,
		timestamp: time.Now(),
	}
	m.sgCountCacheMutex.Unlock()

	return totalSGs, nil
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

				// Get total number of storage groups on the array (cached with TTL)
				totalSGs, err := m.getTotalSGCount(ctx, pmaxClient, arrayID)
				if err != nil {
					m.Logger.WithError(err).WithField("arrayID", arrayID).Warn(
						"failed to get total storage group count, using individual API")
					m.collectSGMetricsLegacy(ctx, pmaxClient, arrayID, sgs, ch)
					return
				}

				requestedSGs := len(sgs)

				// Use bulk API when the monitored fraction exceeds bulkThresholdRatio.
				// Guard against totalSGs == 0 (empty array) to avoid float division producing +Inf.
				useBulk := totalSGs > 0 && requestedSGs > 0 && float64(requestedSGs)/float64(totalSGs) > bulkThresholdRatio
				m.Logger.WithFields(map[string]interface{}{
					"arrayID":      arrayID,
					"requestedSGs": requestedSGs,
					"totalSGs":     totalSGs,
					"useBulk":      useBulk,
				}).Debug("Storage group performance collection strategy")

				if useBulk {
					// Try the bulk GET endpoint first (one API call for all SGs).
					if m.collectSGMetricsBulk(ctx, pmaxClient, arrayID, sgs, ch) {
						return
					}
					// Bulk call failed; fall back to the legacy per-SG POST calls.
					m.Logger.WithField("arrayID", arrayID).Warn(
						"bulk SG performance collection failed, falling back to per-SG collection")
				}
				// Use individual API
				m.collectSGMetricsLegacy(ctx, pmaxClient, arrayID, sgs, ch)
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

// collectSGMetricsBulk tries the single GET /performance-categories/StorageGroup endpoint
// to retrieve all SG metrics in one call. Returns true if it succeeded.
func (m *PerformanceMetrics) collectSGMetricsBulk(
	ctx context.Context,
	pmaxClient metrictypes.PowerMaxClient,
	arrayID string,
	sgs map[string]struct{},
	ch chan<- *metrictypes.StorageGroupPerfMetricsRecord,
) bool {
	bulkResult, err := pmaxClient.GetStorageGroupMetricsBulk(ctx, arrayID)
	if err != nil {
		return false
	}
	if bulkResult == nil {
		return false
	}
	metricsEmitted := 0
	for _, instance := range bulkResult.MetricInstances {
		if _, ok := sgs[instance.ID]; !ok {
			continue
		}
		for _, metric := range instance.Metrics {
			ch <- &metrictypes.StorageGroupPerfMetricsRecord{
				ArrayID:           arrayID,
				StorageGroupID:    instance.ID,
				HostMBReads:       metric.HostMBReads,
				HostMBWritten:     metric.HostMBWritten,
				ReadResponseTime:  metric.ReadResponseTime,
				WriteResponseTime: metric.WriteResponseTime,
				HostReads:         metric.HostReads,
				HostWrites:        metric.HostWrites,
				AvgIOSize:         metric.AvgIOSize,
			}
			metricsEmitted++
		}
	}
	if metricsEmitted == 0 && len(sgs) > 0 {
		m.Logger.WithField("arrayID", arrayID).Warn(
			"bulk SG metrics returned zero records, falling back to per-SG collection")
		return false
	}
	return true
}

// collectSGMetricsLegacy retrieves SG performance metrics using the legacy per-SG POST calls.
// It first calls GetStorageGroupPerfKeys to get timestamps, then GetStorageGroupMetrics per SG.
func (m *PerformanceMetrics) collectSGMetricsLegacy(
	ctx context.Context,
	pmaxClient metrictypes.PowerMaxClient,
	arrayID string,
	sgs map[string]struct{},
	ch chan<- *metrictypes.StorageGroupPerfMetricsRecord,
) {
	timeResult, err := pmaxClient.GetStorageGroupPerfKeys(ctx, arrayID)
	if err != nil {
		m.Logger.WithError(err).WithField("arrayID", arrayID).Warn("cannot query last available time for storage groups in the array")
		return
	}
	// Build a map of SG -> last available time
	storageGroup2LastAvailTime := make(map[string]int64)
	for _, storageGroupInfo := range timeResult.StorageGroupInfos {
		if _, ok := sgs[storageGroupInfo.StorageGroupID]; ok {
			storageGroup2LastAvailTime[storageGroupInfo.StorageGroupID] = storageGroupInfo.LastAvailableDate
			m.Logger.Debugf("last available time of the storage group %s is %d", storageGroupInfo.StorageGroupID, storageGroupInfo.LastAvailableDate)
		}
	}
	for storageGroupID := range sgs {
		lastAvailTime, ok := storageGroup2LastAvailTime[storageGroupID]
		if !ok {
			m.Logger.WithField("storageGroupID", storageGroupID).Warn("last available time for storage group is not found")
			continue
		}
		sgMetrics, sgErr := pmaxClient.GetStorageGroupMetrics(ctx, arrayID, storageGroupID, sgPerfMetricsQuery, lastAvailTime, lastAvailTime)
		if sgErr != nil {
			m.Logger.WithError(sgErr).WithField("storageGroupID ID", storageGroupID).Warn("failed to get storage group metrics")
			continue
		}
		for _, sgResult := range sgMetrics.ResultList.Result {
			ch <- &metrictypes.StorageGroupPerfMetricsRecord{
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
		}
	}
}

// sgPerfMetricsQuery is the set of storage group performance metrics collected per scrape.
var sgPerfMetricsQuery = []string{
	"HostMBReads", "HostMBWritten",
	"ReadResponseTime", "WriteResponseTime", "HostReads", "HostWrites", "AvgIOSize",
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
