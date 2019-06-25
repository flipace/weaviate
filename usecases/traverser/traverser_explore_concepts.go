/*                          _       _
 *__      _____  __ ___   ___  __ _| |_ ___
 *\ \ /\ / / _ \/ _` \ \ / / |/ _` | __/ _ \
 * \ V  V /  __/ (_| |\ V /| | (_| | ||  __/
 *  \_/\_/ \___|\__,_| \_/ |_|\__,_|\__\___|
 *
 * Copyright © 2016 - 2019 Weaviate. All rights reserved.
 * LICENSE: https://github.com/semi-technologies/weaviate/blob/develop/LICENSE.md
 * DESIGN & CONCEPT: Bob van Luijt (@bobvanluijt)
 * CONTACT: hello@semi.technology
 */

package traverser

import (
	"context"
	"fmt"

	"github.com/go-openapi/strfmt"
	"github.com/semi-technologies/weaviate/entities/models"
	"github.com/semi-technologies/weaviate/entities/schema/kind"
)

// ExploreConcepts through unstructured search terms
func (t *Traverser) ExploreConcepts(ctx context.Context,
	principal *models.Principal, params ExploreConceptsParams) ([]VectorSearchResult, error) {

	err := t.authorizer.Authorize(principal, "get", "traversal/*")
	if err != nil {
		return nil, err
	}

	vector, err := t.vectorFromExploreParams(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("vectorize params: %v", err)
	}

	res, err := t.vectorSearcher.VectorSearch(ctx, "concepts", vector, params.Limit)
	if err != nil {
		return nil, fmt.Errorf("vector search: %v", err)
	}

	for i, item := range res {
		res[i].Beacon = beacon(item)
	}

	return res, nil
}

func (t *Traverser) vectorFromExploreParams(ctx context.Context,
	params ExploreConceptsParams) ([]float32, error) {

	vector, err := t.vectorizer.Corpi(ctx, params.Values)
	if err != nil {
		return nil, fmt.Errorf("vectorize keywords: %v", err)
	}

	if params.MoveTo.Force > 0 && len(params.MoveTo.Values) > 0 {
		moveToVector, err := t.vectorizer.Corpi(ctx, params.MoveTo.Values)
		if err != nil {
			return nil, fmt.Errorf("vectorize move to: %v", err)
		}

		afterMoveTo, err := t.vectorizer.MoveTo(vector, moveToVector, params.MoveTo.Force)
		if err != nil {
			return nil, err
		}
		vector = afterMoveTo
	}

	if params.MoveAwayFrom.Force > 0 && len(params.MoveAwayFrom.Values) > 0 {
		moveAwayVector, err := t.vectorizer.Corpi(ctx, params.MoveAwayFrom.Values)
		if err != nil {
			return nil, fmt.Errorf("vectorize move away from: %v", err)
		}

		afterMoveFrom, err := t.vectorizer.MoveAwayFrom(vector, moveAwayVector,
			params.MoveAwayFrom.Force)
		if err != nil {
			return nil, err
		}
		vector = afterMoveFrom
	}

	return vector, nil
}

func beacon(res VectorSearchResult) string {
	return fmt.Sprintf("weaviate://localhost/%ss/%s", res.Kind.Name(), res.ID)

}

// ExploreConceptsParams to do a vector based explore search
type ExploreConceptsParams struct {
	Values       []string
	Limit        int
	MoveTo       ExploreMove
	MoveAwayFrom ExploreMove
}

// ExploreMove moves an existing Search Vector closer (or further away from) a specific other search term
type ExploreMove struct {
	Values []string
	Force  float32
}

// VectorSearchResult contains some info of a concept (kind), but not all. For
// additional info the ID can be used to retrieve the full concept from the
// connector storage
type VectorSearchResult struct {
	ID        strfmt.UUID
	Kind      kind.Kind
	ClassName string
	Score     float32
	Vector    []float32
	Beacon    string
}
