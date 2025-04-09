package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/schema"
	"github.com/mdfriday/hugoverse/internal/domain/content"
	apiFrom "github.com/mdfriday/hugoverse/internal/interfaces/api/form"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/query"
	"github.com/mdfriday/hugoverse/pkg/db"
	"github.com/mdfriday/hugoverse/pkg/form"
	"github.com/mdfriday/hugoverse/pkg/timestamp"
	"log"
	"net/http"
	"strings"
)

func (s *Handler) ApiContentsHandler(res http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	t := q.Get("type")
	if t == "" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	it, ok := s.contentApp.AllTypes()[t]
	if !ok {
		res.WriteHeader(http.StatusNotFound)
		return
	}

	if hide(res, req, it()) {
		return
	}

	count, err := query.Count(req)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	offset, err := query.Offset(req)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	order := query.Order(req)

	opts := db.QueryOptions{
		Count:  count,
		Offset: offset,
		Order:  order,
	}

	_, bb := s.db.Query(t+"__sorted", opts)
	var result []json.RawMessage
	for i := range bb {
		result = append(result, bb[i])
	}

	j, err := s.res.FmtJSON(result...)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	j, err = omit(res, req, it(), j)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	// assert hookable
	get := it()
	hook, ok := get.(content.Hookable)
	if !ok {
		log.Println("[Response] error: Type", t, "does not implement item.Hookable or embed item.Item.")
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	// hook before response
	j, err = hook.BeforeAPIResponse(res, req, j)
	if err != nil {
		log.Println("[Response] error calling BeforeAPIResponse:", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	s.res.Json(res, j)

	// hook after response
	err = hook.AfterAPIResponse(res, req, j)
	if err != nil {
		log.Println("[Response] error calling AfterAPIResponse:", err)
		return
	}
}

func (s *Handler) ContentHandler(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		s.getContent(res, req)
	case http.MethodPost:
		s.postContent(res, req)
	default:
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

func (s *Handler) getContent(res http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	id := q.Get("id")
	t := q.Get("type")
	status := q.Get("status")

	if t == "" || id == "" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	pt, ok := s.contentApp.GetContentCreator(t)
	if !ok {
		res.WriteHeader(http.StatusNotFound)
		return
	}

	post, err := s.contentApp.GetContent(t, id, status)
	if err != nil {
		s.log.Errorf("Error getting content: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	if post == nil {
		res.WriteHeader(http.StatusNotFound)
		s.log.Printf("Content not found: %s %s %s", t, id, status)
		return
	}

	p := pt()

	unmarshal, ok := p.(content.Unmarshalable)
	if ok {
		err = unmarshal.UnmarshalJSON(post)
	} else {
		err = json.Unmarshal(post, p)
	}
	if err != nil {
		s.log.Errorf("Error unmarshalling content: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	if hide(res, req, p) {
		return
	}

	push(res, req, p, post)

	j, err := s.res.FmtJSON(json.RawMessage(post))
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	j, err = omit(res, req, p, j)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	// assert hookable
	get := p
	hook, ok := get.(content.Hookable)
	if !ok {
		log.Println("[Response] error: Type", t, "does not implement item.Hookable or embed item.Item.")
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	// hook before response
	j, err = hook.BeforeAPIResponse(res, req, j)
	if err != nil {
		log.Println("[Response] error calling BeforeAPIResponse:", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	s.res.Json(res, j)

	// hook after response
	err = hook.AfterAPIResponse(res, req, j)
	if err != nil {
		log.Println("[Response] error calling AfterAPIResponse:", err)
		return
	}
}

func (s *Handler) postContent(res http.ResponseWriter, req *http.Request) {
	err := req.ParseMultipartForm(apiFrom.MaxMemory) // maxMemory 4MB
	if err != nil {
		s.log.Errorf("Error parsing multipart form: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	t := req.URL.Query().Get("type")
	if t == "" {
		s.log.Errorf("Attempt to submit without type from %s", req.URL.String())
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	p, found := s.contentApp.AllTypes()[t]
	if !found {
		s.log.Printf("Attempt to submit unknown type: %s from %s", t, req.RemoteAddr)
		res.WriteHeader(http.StatusNotFound)
		return
	}

	post := p()

	cid := req.PostForm.Get("id")
	isUpdating := cid != "-1"
	isCreating := !isUpdating

	if isUpdating {
		ep := p()
		data, err := s.contentApp.GetContent(t, cid, "")
		if err != nil {
			s.log.Errorf("Error getting content: %v with id %s", err, cid)
			res.WriteHeader(http.StatusNotFound)
			return
		}
		err = json.Unmarshal(data, ep)
		if err != nil {
			s.log.Errorf("Error unmarshalling content: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		if sort, ok := ep.(content.Sortable); ok {
			req.PostForm.Set("timestamp", timestamp.TimeToString(sort.Time()))
		}
		if identifier, ok := ep.(content.Identifiable); ok {
			req.PostForm.Set("uuid", identifier.UniqueID().String())
		}
		if slug, ok := ep.(content.Sluggable); ok {
			req.PostForm.Set("slug", slug.ItemSlug())
		}
		if dim, ok := ep.(content.Dimensional); ok {
			req.PostForm.Set("width", fmt.Sprintf("%d", dim.GetWidth()))
			req.PostForm.Set("height", fmt.Sprintf("%d", dim.GetHeight()))
		}
		if update, ok := ep.(content.Updateable); ok {
			err = update.Update(res, req)
			if err != nil {
				s.log.Errorf("Error updating content: %v", err)
				res.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}

	ext, ok := post.(content.Createable)
	if !ok {
		s.log.Printf("Attempt to create non-createable type: %s from %s", t, req.RemoteAddr)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	ts := timestamp.Now()
	if isCreating {
		req.PostForm.Set("timestamp", ts)
	}
	req.PostForm.Set("updated", ts)

	urlPaths, _, err := s.StoreFiles(req)
	if err != nil {
		s.log.Errorf("Error storing files: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	for name, urlPath := range urlPaths {
		req.PostForm.Set(name, urlPath)
	}

	req.PostForm, err = form.Convert(req.PostForm)
	if err != nil {
		s.log.Errorf("Error converting form: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	hook, ok := post.(content.Hookable)
	if !ok {
		s.log.Printf("Attempt to create non-hookable type: %s from %s", t, req.RemoteAddr)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	// Let's be nice and make a proper item for the Hookable methods
	dec := schema.NewDecoder()
	dec.IgnoreUnknownKeys(true)
	dec.SetAliasTag("json")
	err = dec.Decode(post, req.PostForm)
	if err != nil {
		s.log.Printf("Error decoding post form for edit %s handler: %v", t, err)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	err = hook.BeforeAPICreate(res, req)
	if err != nil {
		s.log.Errorf("Error calling BeforeCreate: %v", err)
		return
	}

	err = ext.Create(res, req)
	if err != nil {
		s.log.Errorf("Error calling Accept: %v", err)
		return
	}

	err = hook.BeforeSave(res, req)
	if err != nil {
		s.log.Errorf("Error calling BeforeSave: %v", err)
		return
	}

	// set specifier for db bucket in case content is/isn't Trustable
	var spec string

	// check if the content is Trustable should be auto-approved, if so the
	// content is immediately added to the public content API. If not, then it
	// is added to a "pending" list, only visible to Admins in the CMS and only
	// if the type implements editor.Mergable
	trusted, ok := post.(content.Trustable)
	if ok {
		err := trusted.AutoApprove(res, req)
		if err != nil {
			s.log.Errorf("Error calling AutoApprove: %v", err)
			return
		}
	} else {
		spec = "__pending"
		req.PostForm.Set("status", string(content.Pending))
	}

	req.PostForm.Set("namespace", t)

	if isCreating {
		id, err := s.contentApp.NewContent(t, req.PostForm)
		if err != nil {
			s.log.Errorf("Error calling SetContent: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		cid = id
	} else {
		if err = s.contentApp.UpdateContent(t, req.PostForm); err != nil {
			s.log.Errorf("Error updating content: %s", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// set the target in the context so user can get saved value from db in hook
	ctx := context.WithValue(req.Context(), "target", fmt.Sprintf("%s:%s", t, cid))
	req = req.WithContext(ctx)

	err = hook.AfterSave(res, req)
	if err != nil {
		s.log.Errorf("Error calling AfterSave: %v", err)
		return
	}

	err = hook.AfterAPICreate(res, req)
	if err != nil {
		s.log.Errorf("Error calling AfterAccept: %v", err)
		return
	}

	// create JSON response to send data back to client
	var data map[string]interface{}
	if spec != "" {
		spec = strings.TrimPrefix(spec, "__")
		data = map[string]interface{}{
			"status": spec,
			"type":   t,
		}
	} else {
		spec = "public"
		data = map[string]interface{}{
			"id":     cid,
			"status": spec,
			"type":   t,
		}
	}

	resp := map[string]interface{}{
		"data": []map[string]interface{}{
			data,
		},
	}

	j, err := json.Marshal(resp)
	if err != nil {
		s.log.Errorf("Error marshalling response to JSON: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	_, err = res.Write(j)
	if err != nil {
		s.log.Errorf("Error writing response: %v", err)
		return
	}
}

func (s *Handler) DeleteContentHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	id := req.PostForm.Get("id")
	t := req.PostForm.Get("type")
	status := req.PostForm.Get("status")
	ct := t

	if id == "" || t == "" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	p, ok := s.contentApp.AllContentTypes()[ct]
	if !ok {
		s.log.Printf("Type %s not supported", t)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	post := p()
	hook, ok := post.(content.Hookable)
	if !ok {
		s.log.Printf("Type %s does not implement item.Hookable or embed item.Item.", t)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	data, err := s.contentApp.GetContent(t, id, status)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		s.log.Printf("Error in db.Content %s:%s: %s", t, id, err)
		return
	}
	if data == nil {
		res.WriteHeader(http.StatusNotFound)
		s.log.Printf("Content not found: %s %s %s", t, id, status)
		return
	}

	err = json.Unmarshal(data, post)
	if err != nil {
		log.Println("Error unmarshalling ", t, "=", id, err, " Hooks will be called on a zero-value.")
	}

	reject := req.URL.Query().Get("reject")
	if reject == "true" {
		err = hook.BeforeReject(res, req)
		if err != nil {
			log.Println("Error running BeforeReject method in deleteHandler for:", t, err)
			return
		}
	}

	err = hook.BeforeAdminDelete(res, req)
	if err != nil {
		log.Println("Error running BeforeAdminDelete method in deleteHandler for:", t, err)
		return
	}

	err = hook.BeforeDelete(res, req)
	if err != nil {
		log.Println("Error running BeforeDelete method in deleteHandler for:", t, err)
		return
	}

	err = s.contentApp.DeleteContent(t, id, status)
	if err != nil {
		s.log.Errorf("Error in db.Content %s:%s: %s", t, id, err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := s.adminApp.InvalidateCache(); err != nil {
		s.log.Errorf("Error invalidating cache: %s", err)
	}

	err = hook.AfterDelete(res, req)
	if err != nil {
		s.log.Errorln("Error running AfterDelete method in deleteHandler for:", t, err)
		return
	}

	err = hook.AfterAdminDelete(res, req)
	if err != nil {
		s.log.Errorln("Error running AfterAdminDelete method in deleteHandler for:", t, err)
		return
	}

	if reject == "true" {
		err = hook.AfterReject(res, req)
		if err != nil {
			s.log.Errorln("Error running AfterReject method in deleteHandler for:", t, err)
			return
		}
	}

	res.WriteHeader(http.StatusOK)
}

func (s *Handler) ContentsTagsHandler(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		s.getContentsTags(res, req)
	default:
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

func (s *Handler) getContentsTags(res http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	t := q.Get("type")

	if t == "" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	pt, ok := s.contentApp.GetContentCreator(t)
	if !ok {
		res.WriteHeader(http.StatusNotFound)
		return
	}

	_, ok = pt().(content.Taggable)
	if !ok {
		res.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	posts := s.contentApp.AllContents(t)

	var tags []string
	for _, post := range posts {
		p := pt()
		err = json.Unmarshal(post, p)
		if err != nil {
			s.log.Errorf("Error unmarshalling content: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		tag, ok := p.(content.Taggable)
		if ok {
			tags = append(tags, tag.ItemTags()...)
		}
	}

	tags = removeDuplicates(tags)
	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		s.log.Errorf("Error marshalling tags: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	j, err := s.res.FmtJSON(tagsJSON)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	p := pt()
	j, err = omit(res, req, p, j)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	// assert hookable
	get := p
	hook, ok := get.(content.Hookable)
	if !ok {
		s.log.Errorln("[Response] error: Type", t, "does not implement item.Hookable or embed item.Item.")
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	// hook before response
	j, err = hook.BeforeAPIResponse(res, req, j)
	if err != nil {
		s.log.Errorln("[Response] error calling BeforeAPIResponse:", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	s.res.Json(res, j)

	// hook after response
	err = hook.AfterAPIResponse(res, req, j)
	if err != nil {
		s.log.Errorln("[Response] error calling AfterAPIResponse:", err)
		return
	}
}

// removeDuplicates 去重
func removeDuplicates(tags []string) []string {
	uniqueMap := make(map[string]struct{})
	var uniqueTags []string

	for _, tag := range tags {
		if _, exists := uniqueMap[tag]; !exists {
			uniqueMap[tag] = struct{}{}
			uniqueTags = append(uniqueTags, tag)
		}
	}

	return uniqueTags
}
