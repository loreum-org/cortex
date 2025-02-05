package core

import (
	"log"

	"github.com/blevesearch/bleve"
)

type VectorDatabase struct {
	index bleve.Index
}

func NewVectorDatabase() *VectorDatabase {
	indexMapping := bleve.NewIndexMapping()
	index, err := bleve.NewMemOnly(indexMapping)
	if err != nil {
		log.Fatal(err)
	}
	return &VectorDatabase{index: index}
}

func (vdb *VectorDatabase) StoreData(id, content string) error {
	return vdb.index.Index(id, content)
}

func (vdb *VectorDatabase) Query(query string) ([]string, error) {
	q := bleve.NewMatchQuery(query)
	search := bleve.NewSearchRequest(q)
	result, err := vdb.index.Search(search)
	if err != nil {
		return nil, err
	}

	var hits []string
	for _, hit := range result.Hits {
		hits = append(hits, hit.ID)
	}
	return hits, nil
}
