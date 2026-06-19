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

package member

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/tikv/pd/pkg/utils/testutil"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m, testutil.LeakOptions...)
}

func TestFilterCaughtUpCandidates(t *testing.T) {
	re := require.New(t)
	names := map[uint64]string{1: "pd1", 2: "pd2", 3: "pd3"}
	candidates := []uint64{1, 2, 3}

	// No caught-up info: fall back (nil) so the caller keeps the full set.
	re.Nil(filterCaughtUpCandidates(candidates, names, nil))
	re.Nil(filterCaughtUpCandidates(candidates, names, []string{}))

	// Only the matching candidates are preferred; unknown names are ignored.
	re.Equal([]uint64{2}, filterCaughtUpCandidates(candidates, names, []string{"pd2"}))
	re.ElementsMatch([]uint64{1, 3}, filterCaughtUpCandidates(candidates, names, []string{"pd1", "pd3", "ghost"}))

	// A caught-up member that is not a valid transfer candidate yields nil so
	// the caller falls back rather than transferring to a non-candidate.
	re.Nil(filterCaughtUpCandidates([]uint64{1}, names, []string{"pd2", "pd3"}))
}

func TestSetCaughtUpMembersProvider(t *testing.T) {
	re := require.New(t)
	m := &Member{}

	// Default: no provider configured.
	re.Nil(m.caughtUpMembers())

	m.SetCaughtUpMembersProvider(func() []string { return []string{"pd2"} })
	re.Equal([]string{"pd2"}, m.caughtUpMembers())

	// Clearing the provider returns to the nil default without panicking.
	m.SetCaughtUpMembersProvider(nil)
	re.Nil(m.caughtUpMembers())
}

func TestSetHasCommittedRegionsProvider(t *testing.T) {
	re := require.New(t)
	m := &Member{}

	// Default (no provider): false, so the transfer refusal never triggers in
	// setups that do not wire the signal.
	re.False(m.hasCommittedRegions())

	m.SetHasCommittedRegionsProvider(func() bool { return true })
	re.True(m.hasCommittedRegions())

	m.SetHasCommittedRegionsProvider(func() bool { return false })
	re.False(m.hasCommittedRegions())

	// Clearing the provider returns to the false default without panicking.
	m.SetHasCommittedRegionsProvider(nil)
	re.False(m.hasCommittedRegions())
}

func TestShouldRefuseTransferTarget(t *testing.T) {
	re := require.New(t)

	// Empty/fresh cluster (no syncable history): any target is safe, never refuse.
	re.False(shouldRefuseTransferTarget("pd2", nil, false))
	re.False(shouldRefuseTransferTarget("pd2", []string{}, false))
	re.False(shouldRefuseTransferTarget("pd2", []string{"pd3"}, false))

	// Cluster has region data: refuse only when the target is not caught up.
	re.False(shouldRefuseTransferTarget("pd2", []string{"pd2", "pd3"}, true))
	re.True(shouldRefuseTransferTarget("pd2", []string{"pd3"}, true))
	re.True(shouldRefuseTransferTarget("pd2", nil, true))
}
