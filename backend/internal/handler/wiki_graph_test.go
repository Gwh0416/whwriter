package handler

import (
	"testing"

	"whwriter/backend/internal/model"
)

func TestBuildWikiGraphPayloadIncludesEntityAndLiteralTargets(t *testing.T) {
	itemID := uint(2)
	graph := &model.WikiGraphContext{
		Seeds: []model.WikiEntity{
			{ID: 1, EntityType: model.WikiEntityCharacter, CanonicalName: "林秋"},
		},
		Entities: []model.WikiEntity{
			{ID: 1, EntityType: model.WikiEntityCharacter, CanonicalName: "林秋", Status: model.WikiEntityActive},
			{ID: 2, EntityType: model.WikiEntityItem, CanonicalName: "青铜戒指", Status: model.WikiEntityActive},
		},
		Relations: []model.WikiRelationView{
			{WikiRelation: model.WikiRelation{
				ID:                10,
				SubjectEntityID:   1,
				Predicate:         "持有",
				ObjectEntityID:    &itemID,
				ValidFromChapter:  2,
				ValidUntilChapter: nil,
				SourceType:        "fact",
			}},
			{WikiRelation: model.WikiRelation{
				ID:               11,
				SubjectEntityID:  1,
				Predicate:        "灵石",
				ObjectLiteral:    "120",
				ValidFromChapter: 3,
				SourceType:       "fact",
			}},
		},
	}

	payload := buildWikiGraphPayload(7, 3, graph)
	if payload.BookID != 7 || payload.Chapter != 3 || payload.Depth != 1 {
		t.Fatalf("unexpected graph metadata: %#v", payload)
	}
	if len(payload.SeedIDs) != 1 || payload.SeedIDs[0] != 1 {
		t.Fatalf("unexpected seed ids: %#v", payload.SeedIDs)
	}
	if len(payload.Nodes) != 3 || len(payload.Edges) != 2 {
		t.Fatalf("unexpected graph size: nodes=%#v edges=%#v", payload.Nodes, payload.Edges)
	}
	if !hasGraphNode(payload.Nodes, "literal:11", model.WikiEntityLiteral, "120") {
		t.Fatalf("literal node missing: %#v", payload.Nodes)
	}
	if !hasGraphEdge(payload.Edges, "entity:1", "entity:2", "持有") ||
		!hasGraphEdge(payload.Edges, "entity:1", "literal:11", "灵石") {
		t.Fatalf("graph edges missing: %#v", payload.Edges)
	}
}

func TestSelectWikiOverviewSeedsExcludesDisconnectedEntities(t *testing.T) {
	objectID := uint(2)
	entities := []model.WikiEntity{
		{ID: 1, CanonicalName: "林秋", Status: model.WikiEntityActive},
		{ID: 2, CanonicalName: "青铜戒指", Status: model.WikiEntityActive},
		{ID: 3, CanonicalName: "孤立伏笔", Status: model.WikiEntityActive},
	}
	relations := []model.WikiRelationView{{
		WikiRelation: model.WikiRelation{
			SubjectEntityID: 1,
			ObjectEntityID:  &objectID,
		},
	}}

	seeds := selectWikiOverviewSeeds(entities, relations, 16)
	if len(seeds) != 2 || seeds[0] != 1 || seeds[1] != 2 {
		t.Fatalf("unexpected overview seeds: %#v", seeds)
	}
}

func hasGraphNode(nodes []model.WikiGraphNode, id string, entityType model.WikiEntityType, label string) bool {
	for _, node := range nodes {
		if node.ID == id && node.EntityType == entityType && node.Label == label {
			return true
		}
	}
	return false
}

func hasGraphEdge(edges []model.WikiGraphEdge, source, target, label string) bool {
	for _, edge := range edges {
		if edge.Source == source && edge.Target == target && edge.Label == label {
			return true
		}
	}
	return false
}
