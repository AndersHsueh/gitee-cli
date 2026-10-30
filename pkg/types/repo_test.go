package types

import (
	"encoding/json"
	"testing"
)

func TestRepositoryStarsCountPrefersStargazersCount(t *testing.T) {
	var repo Repository
	raw := []byte(`{"full_name":"andershsueh/gitee-cli","stargazers_count":28,"stars_count":1,"forks_count":2}`)
	if err := json.Unmarshal(raw, &repo); err != nil {
		t.Fatal(err)
	}
	if repo.StarsCount != 28 {
		t.Fatalf("StarsCount = %d, want stargazers_count 28", repo.StarsCount)
	}
	if repo.ForksCount != 2 {
		t.Fatalf("ForksCount = %d, want 2", repo.ForksCount)
	}
}

func TestRepositoryStarsCountLegacyField(t *testing.T) {
	var repo Repository
	raw := []byte(`{"full_name":"o/r","stars_count":7}`)
	if err := json.Unmarshal(raw, &repo); err != nil {
		t.Fatal(err)
	}
	if repo.StarsCount != 7 {
		t.Fatalf("StarsCount = %d, want legacy stars_count 7", repo.StarsCount)
	}
}
