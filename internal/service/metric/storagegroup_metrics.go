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

// ExportStorageGroupMetrics record storage group metrics
// func ExportStorageGroupMetrics(ctx context.Context, s *service.PowerMaxService, pvs []k8s.VolumeInfo) {
// 	start := time.Now()
// 	defer s.TimeSince(start, "ExportArrayCapacityMetrics")
//
// 	if s.MetricsRecorder == nil {
// 		s.Logger.Warn("no MetricsRecorder provided for getting ExportArrayCapacityMetrics")
// 		return
// 	}
//
// 	if s.MaxPowerMaxConnections == 0 {
// 		s.Logger.Debug("Using DefaultMaxPowerMaxConnections")
// 		s.MaxPowerMaxConnections = service.DefaultMaxPowerMaxConnections
// 	}
//
// 	// for range s.pushArrayCapacityStatsMetrics(ctx, s.gatherArrayCapacityStatsMetrics(ctx)) {
// 	// 	// consume the channel until it is empty and closed
// 	// }
// }
