package entity

import (
	"encoding/json"
	"fmt"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/mdfriday/hugoverse/internal/domain/content"
	"github.com/mdfriday/hugoverse/internal/domain/content/repository"
	"github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/pkg/loggers"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type TypeService interface {
	GetContentCreator(name string) (content.Creator, bool)
	AllContentTypeNames() []string
	AllAdminTypeNames() []string
	IsAdminType(name string) bool
}

var cleanupWaitDuration = 5 * time.Minute

type CacheIndex struct {
	bleve.Index

	timer *time.Timer
}

func (ci *CacheIndex) close() error {
	ci.timer.Stop()
	return ci.Index.Close()
}

type Search struct {
	TypeService TypeService

	Repo repository.Repository
	Log  loggers.Logger

	mu         sync.Mutex
	IndicesMap map[string]*CacheIndex
}

func (s *Search) getSearchPath(ns string) string {
	return fmt.Sprintf("%s-%s", s.getSearchDir(ns), ns)
}

func (s *Search) resetDBTimer(index *CacheIndex) {
	if index.timer != nil {
		index.timer.Stop()
	}

	index.timer = time.NewTimer(cleanupWaitDuration)
	go func() {
		<-index.timer.C
		s.cleanupIdleDB(index) // 触发统一的清理函数
	}()
}

func (s *Search) cleanupIdleDB(idleIndex *CacheIndex) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for searchPath, idx := range s.IndicesMap {
		if idx == idleIndex {
			err := idx.close()
			if err != nil {
				s.Log.Errorln("Couldn't close search index.", err)
				return
			}

			delete(s.IndicesMap, searchPath)
			s.Log.Printf("Clean search index %s, %d", searchPath, len(s.IndicesMap))
			return
		}
	}
}

func (s *Search) getSearchIndex(ns string) (bleve.Index, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	searchPath := s.getSearchPath(ns)

	if idx, ok := s.IndicesMap[searchPath]; ok {
		s.resetDBTimer(idx)
		return idx.Index, nil
	}

	index, err := s.mapIndex(ns)
	if err != nil {
		s.Log.Errorln("[search] Setup Error", searchPath, err)
		return nil, err
	}

	ci := &CacheIndex{
		Index: index,
	}
	s.resetDBTimer(ci)

	s.IndicesMap[searchPath] = ci
	return ci.Index, nil
}

// MapIndex creates the mapping for a type and tracks the index to be used within
// the system for adding/deleting/checking data
func (s *Search) mapIndex(typeName string) (bleve.Index, error) {
	it, ok := s.TypeService.GetContentCreator(typeName)
	if !ok {
		return nil, fmt.Errorf("[search] MapIndex Error: Failed to MapIndex for %s, type doesn't exist", typeName)
	}
	sc, ok := it().(content.Searchable)
	if !ok {
		return nil, fmt.Errorf("[search] MapIndex Error: Item type %s doesn't implement search.Searchable", typeName)
	}

	// skip setting or using index for types that shouldn't be indexed
	if !sc.IndexContent() {
		s.Log.Warnf("[search] Index not created for %s\n", typeName)
		return nil, nil
	}

	mapping, err := sc.SearchMapping()
	if err != nil {
		return nil, err
	}

	idxName := typeName + ".index"
	var idx bleve.Index

	searchPath := s.getSearchDir(typeName)

	err = os.MkdirAll(searchPath, os.ModeDir|os.ModePerm)
	if err != nil {
		return nil, err
	}

	idxPath := filepath.Join(searchPath, idxName)
	if _, err = os.Stat(idxPath); os.IsNotExist(err) {
		idx, err = bleve.New(idxPath, mapping)
		if err != nil {
			return nil, err
		}
		idx.SetName(idxName)
		s.Log.Debugf("[search] Index new created for %s\n", typeName)
	} else {
		idx, err = bleve.Open(idxPath)
		if err != nil {
			return nil, err
		}
		s.Log.Debugf("[search] Index open created for %s\n", typeName)
	}

	return idx, nil
}

// TypeQuery conducts a search and returns a set of Ponzu "targets", Type:ID pairs,
// and an error. If there is no search index for the typeName (Type) provided,
// db.ErrNoIndex will be returned as the error
func (s *Search) TypeQuery(typeName, query string, count, offset int) ([]content.Identifier, error) {
	idx, err := s.getSearchIndex(typeName)
	if err != nil {
		s.Log.Errorln("Index for type ", typeName, " not found", err.Error())
		return nil, content.ErrNoIndex
	}

	s.Log.Infoln("TypeQuery: ", query)
	q := bleve.NewQueryStringQuery(query)
	req := bleve.NewSearchRequestOptions(q, count, offset, false)
	res, err := idx.Search(req)
	if err != nil {
		return nil, err
	}

	var results []content.Identifier
	for _, hit := range res.Hits {
		results = append(results, valueobject.CreateIndex(hit.ID))
	}

	return results, nil
}

func (s *Search) TermQuery(typeName string, keyValues map[string]string, count, offset int) ([]content.Identifier, error) {
	// 获取索引
	idx, err := s.getSearchIndex(typeName)
	if err != nil {
		s.Log.Debugln("Index for type ", typeName, " not found", err.Error())
		return nil, content.ErrNoIndex
	}

	var termQueries []query.Query
	for key, value := range keyValues {
		tq := bleve.NewTermQuery(value)
		tq.SetField(key)

		termQueries = append(termQueries, tq)
	}

	// 将查询组合成一个 ConjunctionQuery
	finalQuery := bleve.NewConjunctionQuery(termQueries...)

	// 创建搜索请求
	req := bleve.NewSearchRequestOptions(finalQuery, count, offset, false)

	// 执行搜索
	res, err := idx.Search(req)
	if err != nil {
		return nil, err
	}

	// 处理搜索结果
	var results []content.Identifier
	for _, hit := range res.Hits {
		results = append(results, valueobject.CreateIndex(hit.ID))
	}

	return results, nil
}

// UpdateIndex sets data into a content type's search index at the given
// identifier
func (s *Search) UpdateIndex(ns, id string, data []byte) error {
	idx, err := s.getSearchIndex(ns)
	if err != nil {
		return err
	}

	// unmarshal json to struct, error if not registered
	it, ok := s.TypeService.GetContentCreator(ns)
	if !ok {
		return fmt.Errorf("[search] UpdateIndex Error: type '%s' doesn't exist", ns)
	}

	p := it()
	err = json.Unmarshal(data, &p)
	if err != nil {
		return err
	}

	// add data to search index
	i := valueobject.NewIndex(ns, id)
	err = idx.Index(i.String(), p)

	return err
}

// DeleteIndex removes data from a content type's search index at the
// given identifier
func (s *Search) DeleteIndex(id string) error {
	target := strings.Split(id, ":")
	ns := target[0]

	idx, err := s.getSearchIndex(ns)
	if err != nil {
		return err
	}

	return idx.Delete(id)
}

func (s *Search) getSearchDir(ns string) string {
	if s.TypeService.IsAdminType(ns) {
		return s.adminSearchDir()
	}
	return s.userSearchDir()
}

func (s *Search) userSearchDir() string {
	return filepath.Join(s.Repo.UserDataDir(), "Search")
}

func (s *Search) adminSearchDir() string {
	return filepath.Join(s.Repo.AdminDataDir(), "Search")
}
