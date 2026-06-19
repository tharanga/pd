// Copyright 2026 TiKV Project Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package syncer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCaughtUpMembers(t *testing.T) {
	re := require.New(t)

	// A leader with no bound streams reports nobody.
	empty := &RegionSyncer{}
	empty.mu.streams = map[string]*regionSyncStream{}
	re.Nil(empty.CaughtUpMembers())

	served := newRegionSyncStream(&testServerStream{}, 10)
	served.markHistoryServed()
	unserved := newRegionSyncStream(&testServerStream{}, 10)

	syncer := &RegionSyncer{}
	syncer.mu.streams = map[string]*regionSyncStream{
		"pd-served":   served,
		"pd-unserved": unserved,
	}

	// Only a stream that has finished historical catch-up is reported.
	re.Equal([]string{"pd-served"}, syncer.CaughtUpMembers())

	// Once the other stream completes its historical catch-up it joins.
	unserved.markHistoryServed()
	re.ElementsMatch([]string{"pd-served", "pd-unserved"}, syncer.CaughtUpMembers())
}

func TestHasSyncableHistory(t *testing.T) {
	re := require.New(t)

	syncer := &RegionSyncer{history: newTestHistoryBuffer(8)}
	// A fresh history buffer has distributed nothing.
	re.False(syncer.HasSyncableHistory())

	// Recording a region advances the history index, so there is now syncable
	// history a follower could be behind on.
	syncer.history.record(newHistoryBufferTestRegion(1))
	re.True(syncer.HasSyncableHistory())
}
