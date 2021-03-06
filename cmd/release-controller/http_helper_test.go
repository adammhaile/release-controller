package main

import (
	"reflect"
	"sort"
	"testing"

	imagev1 "github.com/openshift/api/image/v1"
	"k8s.io/apimachinery/pkg/util/diff"
)

func Test_calculateReleaseUpgrades(t *testing.T) {
	tests := []struct {
		name    string
		release *Release
		tags    []*imagev1.TagReference
		graph   func() *UpgradeGraph
		want    *ReleaseUpgrades
		wantFn  func() *ReleaseUpgrades
	}{
		{
			tags: []*imagev1.TagReference{},
			graph: func() *UpgradeGraph {
				g := NewUpgradeGraph()
				return g
			},
			want: &ReleaseUpgrades{
				Width: 0,
				Tags:  []ReleaseTagUpgrade{},
			},
		},
		{
			tags: []*imagev1.TagReference{
				{Name: "4.0.1"},
				{Name: "4.0.0"},
			},
			graph: func() *UpgradeGraph {
				g := NewUpgradeGraph()
				return g
			},
			want: &ReleaseUpgrades{
				Width: 0,
				Tags: []ReleaseTagUpgrade{
					{},
					{},
				},
			},
		},
		{
			tags: []*imagev1.TagReference{
				{Name: "4.0.1"},
				{Name: "4.0.0"},
				{Name: "4.0.0-9"},
			},
			graph: func() *UpgradeGraph {
				g := NewUpgradeGraph()
				g.Add("4.0.0", "4.0.1", UpgradeResult{State: releaseVerificationStateFailed, Url: "https://test.com/1"})
				return g
			},
			wantFn: func() *ReleaseUpgrades {
				internal0 := []UpgradeHistory{{From: "4.0.0", To: "4.0.1", Success: 0, Failure: 1, Total: 1}}
				u := &ReleaseUpgrades{
					Width: 1,
					Tags: []ReleaseTagUpgrade{
						{
							Internal: internal0,
							Visual: []ReleaseTagUpgradeVisual{
								{Begin: &internal0[0]},
							},
						},
						{
							Visual: []ReleaseTagUpgradeVisual{
								{End: &internal0[0]},
							},
						},
						{},
					},
				}
				return u
			},
		},
		{
			tags: []*imagev1.TagReference{
				{Name: "4.0.5"},
				{Name: "4.0.4"},
				{Name: "4.0.3"},
				{Name: "4.0.2"},
				{Name: "4.0.1"},
			},
			graph: func() *UpgradeGraph {
				g := NewUpgradeGraph()
				g.Add("4.0.4", "4.0.5", UpgradeResult{State: releaseVerificationStateFailed, Url: "https://test.com/1"})
				g.Add("4.0.3", "4.0.5", UpgradeResult{State: releaseVerificationStateSucceeded, Url: "https://test.com/2"})
				g.Add("4.0.0", "4.0.2", UpgradeResult{State: releaseVerificationStateSucceeded, Url: "https://test.com/2"})
				return g
			},
			wantFn: func() *ReleaseUpgrades {
				internal0 := []UpgradeHistory{
					{From: "4.0.4", To: "4.0.5", Success: 0, Failure: 1, Total: 1},
					{From: "4.0.3", To: "4.0.5", Success: 1, Failure: 0, Total: 1},
				}
				u := &ReleaseUpgrades{
					Width: 2,
					Tags: []ReleaseTagUpgrade{
						{
							Internal: internal0,
							Visual: []ReleaseTagUpgradeVisual{
								{Begin: &internal0[0]},
								{Begin: &internal0[1]},
							},
						},
						{
							Visual: []ReleaseTagUpgradeVisual{
								{End: &internal0[0]},
								{Current: &internal0[1]},
							},
						},
						{
							Visual: []ReleaseTagUpgradeVisual{
								{},
								{End: &internal0[1]},
							},
						},
						{
							External: []UpgradeHistory{{From: "4.0.0", To: "4.0.2", Success: 1, Total: 1}},
						},
						{},
					},
				}
				return u
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantFn != nil {
				tt.want = tt.wantFn()
			}
			if tt.release == nil {
				tt.release = &Release{
					Config: &ReleaseConfig{},
				}
			}
			if got := calculateReleaseUpgrades(tt.release, tt.tags, tt.graph()); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%s", diff.ObjectReflectDiff(tt.want, got))
			}
		})
	}
}

func TestSemanticVersions_Tags(t *testing.T) {
	tests := []struct {
		name string
		v    SemanticVersions
		want []*imagev1.TagReference
	}{
		{
			v: NewSemanticVersions([]*imagev1.TagReference{
				{Name: "4.0.0"}, {Name: "4.0.1"}, {Name: "4.0.0-2"}, {Name: "4.0.0-1-a"},
			}),
			want: []*imagev1.TagReference{
				{Name: "4.0.1"}, {Name: "4.0.0"}, {Name: "4.0.0-1-a"}, {Name: "4.0.0-2"},
			},
		},
		{
			v: NewSemanticVersions([]*imagev1.TagReference{
				{Name: "4.0.0-0.9"}, {Name: "4.0.0-0.2"}, {Name: "4.0.0-0.2.a"},
			}),
			want: []*imagev1.TagReference{
				{Name: "4.0.0-0.9"}, {Name: "4.0.0-0.2.a"}, {Name: "4.0.0-0.2"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sort.Sort(tt.v)
			if got := tt.v.Tags(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SemanticVersions.Tags() = %v, want %v", tagNames(got), tagNames(tt.want))
			}
		})
	}
}
