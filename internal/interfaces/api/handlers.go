package api

import (
	"fmt"
	"github.com/mdfriday/hugoverse/internal/application"
	"net/http"
)

func (s *Server) registerContentHandler() {
	s.mux.HandleFunc("/api/contents", s.wrapContentHandler(s.handler.ApiContentsHandler))
	s.mux.HandleFunc("/api/content", s.wrapContentHandler(
		s.content.Handle(s.handler.ContentHandler)))
	s.mux.HandleFunc("/api/content/delete", s.wrapContentHandler(
		s.content.Handle(s.handler.DeleteContentHandler)))

	s.mux.HandleFunc("/api/hash", s.wrapContentHandler(s.handler.HashHandler))

	s.mux.HandleFunc("/api/search", s.wrapContentHandler(s.handler.SearchContentHandler))
	s.mux.HandleFunc("/api/search2", s.wrapContentHandler(s.handler.SearchContentHandler2))

	s.mux.HandleFunc("/api/preview", s.wrapContentHandler(s.handler.PreviewContentHandler))
	s.mux.HandleFunc("/api/build", s.wrapContentHandler(s.handler.BuildContentHandler))
	s.mux.HandleFunc("/api/deploy", s.wrapContentHandler(s.handler.DeployContentHandler))
	s.mux.HandleFunc("/api/deploy/progress", s.wrapContentHandler(s.handler.DeployProgressHandler))
}

func (s *Server) wrapContentHandler(handler http.HandlerFunc) http.HandlerFunc {
	return s.record.Collect(
		s.cors.Handle(
			s.comp.Gzip(
				s.db.Open(
					s.auth.Check(handler)))))
}

func (s *Server) registerUserHandler() {
	s.mux.HandleFunc("/api/user", s.record.Collect(s.cors.Handle(s.content.Handle(s.handler.UserRegisterHandler))))
	s.mux.HandleFunc("/api/login", s.record.Collect(s.cors.Handle(s.content.Handle(s.handler.UserLoginHandler))))
}

func (s *Server) wrapAdminHandler(handler http.HandlerFunc) http.HandlerFunc {
	return s.db.Open(s.auth.CheckWithRedirect(handler))
}

func (s *Server) registerAdminHandler() {
	s.mux.HandleFunc("/admin", s.wrapAdminHandler(s.handler.AdminHandler))

	s.mux.HandleFunc("/admin/login", s.handler.LoginHandler)
	s.mux.HandleFunc("/admin/logout", s.handler.LogoutHandler)

	s.mux.HandleFunc("/admin/configure", s.wrapAdminHandler(s.handler.ConfigHandler))
	s.mux.HandleFunc("/admin/configure/users", s.wrapAdminHandler(s.handler.UserConfigHandler))

	s.mux.HandleFunc("/admin/contents", s.wrapAdminHandler(s.handler.ContentsHandler))
	s.mux.HandleFunc("/admin/contents/search", s.wrapAdminHandler(s.handler.SearchHandler))

	s.mux.HandleFunc("/admin/edit", s.wrapAdminHandler(s.handler.EditHandler))
	s.mux.HandleFunc("/admin/edit/delete", s.wrapAdminHandler(s.handler.DeleteHandler))
	s.mux.HandleFunc("/admin/edit/approve", s.wrapAdminHandler(s.handler.ApproveContentHandler))

	s.mux.HandleFunc("/admin/uploads", s.wrapAdminHandler(s.handler.UploadContentsHandler))
	s.mux.HandleFunc("/admin/uploads/search", s.wrapAdminHandler(s.handler.UploadSearchHandler))
	s.mux.HandleFunc("/admin/edit/upload", s.wrapAdminHandler(s.handler.EditUploadHandler))
	s.mux.HandleFunc("/admin/edit/upload/delete", s.wrapAdminHandler(s.handler.DeleteUploadHandler))

	s.mux.HandleFunc("/admin/init", s.handler.InitHandler)

	s.mux.Handle("/admin/static/", s.cache.Control(
		http.FileServer(adminStaticDir())))

	uploadsDir := application.UploadDir()
	s.mux.Handle("/api/uploads/", s.record.Collect(s.cors.Handle(s.cache.Control(
		http.StripPrefix("/api/uploads/",
			http.FileServer(restrict(http.Dir(uploadsDir))))))))

	previewPath := fmt.Sprintf("/%s/", application.PreviewFolder())
	s.mux.Handle(previewPath, s.record.Collect(s.cors.Handle(s.cache.Control(
		http.StripPrefix(previewPath,
			http.FileServer(restrict(http.Dir(application.PreviewDir()))))))))

}
