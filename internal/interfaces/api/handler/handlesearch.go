package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/mapping"
	"github.com/blevesearch/bleve/search/query"
	"github.com/mdfriday/hugoverse/internal/domain/content"
	"github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/pkg/editor"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

var indexMapping *mapping.IndexMappingImpl
var exampleIndex bleve.Index
var err error

func ExampleNew() {
	indexMapping = bleve.NewIndexMapping()
	indexMapping.StoreDynamic = false

	//docMapping := bleve.NewDocumentMapping()
	//
	//keywordFieldMapping := bleve.NewTextFieldMapping()
	//keywordFieldMapping.Analyzer = keyword.Name
	//
	//// 为不同字段设置映射
	//docMapping.AddFieldMappingsAt("Name", keywordFieldMapping)
	//docMapping.AddFieldMappingsAt("Code", keywordFieldMapping)
	//
	//// 将文档映射添加到索引映射
	//indexMapping.AddDocumentMapping("document", docMapping)

	exampleIndex, err = bleve.New("/Users/sunwei/github/.mdfriday/hugoverse/tmp", indexMapping)
	if err != nil {
		panic(err)
	}
	count, err := exampleIndex.DocCount()
	if err != nil {
		panic(err)
	}

	fmt.Println(count)
	// Output:
	// 0
}

type dodo struct {
	Name string
	Code string
	Hash string
}

func (d *dodo) hash() {
	// 拼接 Name 和 Code
	data := d.Name + d.Code
	// 使用 SHA-256 哈希函数
	hash := sha256.Sum256([]byte(data))
	// 返回哈希值
	d.Hash = hex.EncodeToString(hash[:])
}

func newDodo(name, code string) *dodo {
	d := &dodo{Name: name, Code: code}
	d.hash()
	return d
}

var data = newDodo("app..mdfriday.com", "Untitled Friday Site 299")
var data2 = newDodo("app..mdfriday.com", "Untitled Friday Site 199")

func ExampleIndex_indexing() {
	// index some data
	err = exampleIndex.Index("document id 1", &data)
	if err != nil {
		panic(err)
	}
	err = exampleIndex.Index("document id 2", &data2)
	if err != nil {
		panic(err)
	}

	// 2 documents have been indexed
	count, err := exampleIndex.DocCount()
	if err != nil {
		panic(err)
	}

	fmt.Println(count)
	// Output:
	// 2
}

func (s *Handler) SearchContentHandler2(res http.ResponseWriter, req *http.Request) {
	err = os.RemoveAll("/Users/sunwei/github/.mdfriday/hugoverse/tmp")
	if err != nil {
		panic(err)
	}

	ExampleNew()
	ExampleIndex_indexing()

	fmt.Println("hash: ", data2.Hash)
	var keyValues = map[string]string{"Hash": data2.Hash}

	var termQueries []query.Query
	for key, value := range keyValues {
		tq := bleve.NewTermQuery(value)
		tq.SetField(key)

		termQueries = append(termQueries, tq)
	}

	// 将查询组合成一个 ConjunctionQuery
	finalQuery := bleve.NewConjunctionQuery(termQueries...)
	q := bleve.NewSearchRequestOptions(finalQuery, 10, 0, false)

	indices, err := exampleIndex.Search(q)
	if err != nil {
		panic(err)
	}
	fmt.Println("Search Results for conjunction:", len(indices.Hits))
	for _, index := range indices.Hits {
		fmt.Println(index)
	}

	//// Perform search for "en99"
	//query := bleve.NewTermQuery("en99")
	//query.SetField("Code")
	//searchRequest := bleve.NewSearchRequest(query)
	//searchResults, err := exampleIndex.Search(searchRequest)
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Println("Search Results for en99:", len(searchResults.Hits)) // Expected output: 1
	//
	//// Perform search for "en-100"
	//query2 := bleve.NewMatchQuery("Untitled Friday Site 199")
	//query2.SetField("Code")
	//searchRequest2 := bleve.NewSearchRequest(query2)
	//searchResults2, err := exampleIndex.Search(searchRequest2)
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Println("Search Results for en-100:", len(searchResults2.Hits))

	qr, err := s.contentApp.Search.TermQuery("Language",
		map[string]string{"hash": valueobject.Hash([]string{"English888", "en888"})}, 10, 0)
	if errors.Is(err, content.ErrNoIndex) {
		res.WriteHeader(http.StatusNotFound)
		return
	}

	for _, index := range qr {
		fmt.Println(index.ID(), index.ContentType())
	}

	fmt.Println("???---???", len(qr))

	res.WriteHeader(http.StatusOK)
}

func (s *Handler) SearchContentHandler(res http.ResponseWriter, req *http.Request) {
	qs := req.URL.Query()
	t := qs.Get("type")
	if t == "" {
		s.log.Errorf("Type must be set")
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	it, ok := s.contentApp.AllTypes()[t]
	if !ok {
		s.log.Errorf("Type %s not found", t)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	if hide(res, req, it()) {
		return
	}

	q, err := url.QueryUnescape(qs.Get("q"))
	s.log.Debugln("Query: " + q)
	if err != nil {
		s.log.Errorf("Error unescaping query: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	// q must be set
	if q == "" {
		s.log.Errorf("Query must be set")
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	count, err := strconv.Atoi(qs.Get("count")) // int: determines number of posts to return (10 default, -1 is all)
	if err != nil {
		if qs.Get("count") == "" {
			count = 10
		} else {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	offset, err := strconv.Atoi(qs.Get("offset")) // int: multiplier of count for pagination (0 default)
	if err != nil {
		if qs.Get("offset") == "" {
			offset = 0
		} else {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// execute search for query provided, if no index for type send 404
	indices, err := s.contentApp.Search.TypeQuery(t, q, count, offset)
	if errors.Is(err, content.ErrNoIndex) {
		s.log.Errorf("Index for type %s not found", t)
		res.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		s.log.Errorf("Error searching for type %s: %v", t, err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	// respond with json formatted results
	bb, err := s.contentApp.GetContents(indices)
	if err != nil {
		s.log.Errorf("Error getting content: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	// if we have matches, push the first as its matched by relevance
	if len(bb) > 0 {
		push(res, req, it(), bb[0])
	}

	var result []json.RawMessage
	for i := range bb {
		result = append(result, bb[i])
	}

	j, err := s.res.FmtJSON(result...)
	if err != nil {
		s.log.Errorf("Error formatting json: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	j, err = omit(res, req, it(), j)
	if err != nil {
		s.log.Errorf("Error formatting json: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	s.res.Json(res, j)
}

func (s *Handler) SearchHandler(res http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	t := q.Get("type")
	query := q.Get("q")
	status := q.Get("status")
	var specifier string

	if t == "" || query == "" {
		http.Redirect(res, req, req.URL.Scheme+req.URL.Host+"/admin", http.StatusFound)
		return
	}

	if status == "pending" {
		specifier = "__" + status
	}

	posts := s.db.AllContent(t + specifier)
	b := &bytes.Buffer{}
	pt, ok := s.contentApp.AllContentTypes()[t]
	if !ok {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	post := pt()

	p := post.(editor.Editable)

	html := `<div class="col s9 card">		
					<div class="card-content">
					<div class="row">
					<div class="card-title col s7">` + t + ` Results</div>	
					<form class="col s4" action="/admin/contents/search" method="get">
						<div class="input-field post-search inline">
							<label class="active">Search:</label>
							<i class="right material-icons search-icon">search</i>
							<input class="search" name="q" type="text" placeholder="Within all ` + t + ` fields" class="search"/>
							<input type="hidden" name="type" value="` + t + `" />
							<input type="hidden" name="status" value="` + status + `" />
						</div>
                    </form>	
					</div>
					<ul class="posts row">`

	for i := range posts {
		// skip posts that don't have any matching search criteria
		match := strings.ToLower(query)
		all := strings.ToLower(string(posts[i]))
		if !strings.Contains(all, match) {
			continue
		}

		err := json.Unmarshal(posts[i], &p)
		if err != nil {
			s.log.Printf("Error unmarshal search result json into %s with err: %v, content: %v", t, err, posts[i])

			post := `<li class="col s12">Error decoding data. Possible file corruption.</li>`
			_, err = b.Write([]byte(post))
			if err != nil {
				s.log.Errorf("[admin] Error: %v", err)

				res.WriteHeader(http.StatusInternalServerError)
				errView, err := s.adminView.Error500()
				if err != nil {
					s.log.Errorf("[admin] Error: %v", err)
				}

				res.Write(errView)
				return
			}
			continue
		}

		post := adminPostListItem(p, t, status)
		_, err = b.Write([]byte(post))
		if err != nil {
			s.log.Errorf("[admin] Error: %v", err)

			res.WriteHeader(http.StatusInternalServerError)
			errView, err := s.adminView.Error500()
			if err != nil {
				s.log.Errorf("[admin] Error: %v", err)
			}

			res.Write(errView)
			return
		}
	}

	_, err := b.WriteString(`</ul></div></div>`)
	if err != nil {
		s.log.Errorf("[admin] Error: %v", err)

		res.WriteHeader(http.StatusInternalServerError)
		errView, err := s.adminView.Error500()
		if err != nil {
			s.log.Errorf("[admin] Error: %v", err)
		}

		res.Write(errView)
		return
	}

	script := `
	<script>
		$(function() {
			var del = $('.quick-delete-post.__ponzu span');
			del.on('click', function(e) {
				if (confirm("[Ponzu] Please confirm:\n\nAre you sure you want to delete this post?\nThis cannot be undone.")) {
					$(e.target).parent().submit();
				}
			});
		});

		// disable link from being clicked if parent is 'disabled'
		$(function() {
			$('ul.pagination li.disabled a').on('click', function(e) {
				e.preventDefault();
			});
		});
	</script>
	`

	btn := `<div class="col s3">
		<a href="/admin/edit?type=` + t + `" class="btn new-post waves-effect waves-light">
			New ` + t + `
		</a>`

	html += b.String() + script + btn + `</div></div>`

	adminView, err := s.adminView.SubView([]byte(html))
	if err != nil {
		s.log.Errorf("[admin] Error: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "text/html")
	res.Write(adminView)
}
