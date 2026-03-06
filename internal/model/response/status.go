//
// Copyright (C) 2026 Tim Sleptsov
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package response

import "time"

type TableStatus struct {
	Name                  string     `json:"name"`
	Rows                  int64      `json:"rows"`
	BytesOnDisk           int64      `json:"bytes_on_disk"`
	BytesOnDiskHuman      string     `json:"bytes_on_disk_human"`
	DataUncompressedBytes int64      `json:"data_uncompressed_bytes"`
	DataUncompressedHuman string     `json:"data_uncompressed_human"`
	CompressionRatio      float64    `json:"compression_ratio"`
	MinTimestamp          *time.Time `json:"min_timestamp,omitempty"`
	MaxTimestamp          *time.Time `json:"max_timestamp,omitempty"`
}

type DBStatus struct {
	TotalRows       int64         `json:"total_rows"`
	TotalBytes      int64         `json:"total_bytes"`
	TotalBytesHuman string        `json:"total_bytes_human"`
	Tables          []TableStatus `json:"tables"`
}
