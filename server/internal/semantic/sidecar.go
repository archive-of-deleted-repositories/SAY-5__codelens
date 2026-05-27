package semantic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/SAY-5/codelens/server/internal/store"
)

// Sidecar is a provider backed by the Python semantic service. It posts the
// target and candidate symbols and receives a ranked list. The sidecar itself
// runs behind its own provider seam, so the whole pipeline stays offline in CI.
type Sidecar struct {
	BaseURL string
	Lister  CandidateLister
	Client  *http.Client
}

type rankRequest struct {
	Target     symbolDTO   `json:"target"`
	Candidates []symbolDTO `json:"candidates"`
	Limit      int         `json:"limit"`
}

type symbolDTO struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type rankResult struct {
	Results []struct {
		ID    int64   `json:"id"`
		Score float64 `json:"score"`
	} `json:"results"`
}

// Related calls the sidecar and maps its scored ids back to store symbols.
func (s Sidecar) Related(ctx context.Context, target *store.Symbol, limit int) ([]Related, error) {
	cands, err := s.Lister.AllSymbols(ctx)
	if err != nil {
		return nil, err
	}
	byID := map[int64]store.Symbol{}
	dtos := make([]symbolDTO, 0, len(cands))
	for _, c := range cands {
		byID[c.ID] = c
		dtos = append(dtos, symbolDTO{ID: c.ID, Name: c.Name, Kind: c.Kind})
	}
	body, err := json.Marshal(rankRequest{
		Target:     symbolDTO{ID: target.ID, Name: target.Name, Kind: target.Kind},
		Candidates: dtos,
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.BaseURL+"/rank", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sidecar status %d", resp.StatusCode)
	}
	var rr rankResult
	if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
		return nil, err
	}
	out := make([]Related, 0, len(rr.Results))
	for _, r := range rr.Results {
		if sym, ok := byID[r.ID]; ok {
			s := sym
			out = append(out, Related{Symbol: &s, Score: r.Score})
		}
	}
	return out, nil
}
