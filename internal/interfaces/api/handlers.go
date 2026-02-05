package api

import (
	"fmt"
	"net/http"

	"github.com/mdfriday/hugoverse/internal/application"
)

func (s *Server) registerLicenseHandler() {
	// Register License API endpoints
	// 所有 License API 都需要认证 (需要 TOKEN)

	// License 激活 (需要认证)
	s.mux.HandleFunc("/api/license/activate", s.wrapContentHandler(s.handler.ActivateLicenseHandler))

	// License 信息查询 (需要认证)
	s.mux.HandleFunc("/api/license/info", s.wrapContentHandler(s.handler.GetLicenseInfoHandler))

	// 设备和 IP 管理 (需要认证)
	s.mux.HandleFunc("/api/license/devices", s.wrapContentHandler(s.handler.GetDevicesHandler))
	s.mux.HandleFunc("/api/license/ips", s.wrapContentHandler(s.handler.GetIPsHandler))
	s.mux.HandleFunc("/api/license/device/block", s.wrapContentHandler(s.handler.BlockDeviceHandler))
	s.mux.HandleFunc("/api/license/ip/block", s.wrapContentHandler(s.handler.BlockIPHandler))

	// Sync 和 Publish 服务 (需要认证)
	s.mux.HandleFunc("/api/license/sync", s.wrapContentHandler(s.handler.GetSyncInfoHandler))
	s.mux.HandleFunc("/api/license/publish", s.wrapContentHandler(s.handler.GetPublishInfoHandler))

	// 硬盘用量查询 (需要认证)
	s.mux.HandleFunc("/api/license/disks", s.wrapContentHandler(s.handler.GetDisksHandler))

	// 使用量汇总查询 (需要认证)
	s.mux.HandleFunc("/api/license/usage", s.wrapContentHandler(s.handler.GetUsageHandler))
	s.mux.HandleFunc("/api/license/usage/reset", s.wrapContentHandler(s.handler.ResetUsageHandler))

	s.mux.HandleFunc("/api/mdf/preview", s.wrapContentHandler(s.handler.MDFPreviewHandler))
	s.mux.HandleFunc("/api/mdf/preview/deploy", s.wrapContentHandler(s.handler.DeployMDFridayPreviewHandler))

	// SubDomain 管理
	s.mux.HandleFunc("/api/license/subdomain", s.wrapContentHandler(s.handler.GetSubDomainHandler))
	s.mux.HandleFunc("/api/license/subdomain/check", s.wrapContentHandler(s.handler.CheckSubDomainHandler))
	s.mux.HandleFunc("/api/license/subdomain/update", s.wrapContentHandler(s.handler.UpdateSubDomainHandler))

	// 自定义域名管理
	s.mux.HandleFunc("/api/license/domain/check", s.wrapContentHandler(s.handler.CheckDomainHandler))
	s.mux.HandleFunc("/api/license/domain/add", s.wrapContentHandler(s.handler.AddDomainHandler))
	s.mux.HandleFunc("/api/license/domain/remove", s.wrapContentHandler(s.handler.RemoveDomainHandler))
	s.mux.HandleFunc("/api/license/domains", s.wrapContentHandler(s.handler.GetDomainsHandler))
	s.mux.HandleFunc("/api/license/domain/https-status", s.wrapContentHandler(s.handler.DomainSSLStatusHandler))
}

func (s *Server) registerContentHandler() {
	s.mux.HandleFunc("/api/contents", s.wrapContentHandler(s.handler.ApiContentsHandler))
	s.mux.HandleFunc("/api/content", s.wrapContentHandler(
		s.content.Handle(s.handler.ContentHandler)))
	s.mux.HandleFunc("/api/content/delete", s.wrapContentHandler(
		s.content.Handle(s.handler.DeleteContentHandler)))
	s.mux.HandleFunc("/api/content/tags", s.wrapContentHandler(
		s.content.Handle(s.handler.ContentsTagsHandler)))

	s.mux.HandleFunc("/api/hash", s.wrapContentHandler(s.handler.HashHandler))
	s.mux.HandleFunc("/api/signature", s.wrapContentHandler(s.handler.SignatureHandler))
	s.mux.HandleFunc("/api/cta/submit", s.wrapSignatureHandler(s.handler.CTAHandler))

	s.mux.HandleFunc("/api/counter", s.wrapCounterHandler(s.handler.CounterHandler))

	s.mux.HandleFunc("/api/images", s.wrapPublicHandler(s.handler.ImagesHandler))
	s.mux.HandleFunc("/api/image", s.wrapPublicHandler(s.handler.ImageHandler))
	s.mux.HandleFunc("/api/image/search", s.wrapPublicHandler(s.handler.SearchContentHandler))
	s.mux.HandleFunc("/api/image/tags", s.wrapPublicHandler(s.handler.ContentsTagsHandler))
	s.mux.HandleFunc("/image/{size:[0-9]+}{extension:(?:\\..*)?}",
		s.wrapPublicHandler(s.handler.ImageRandomHandler))
	s.mux.HandleFunc("/image/{width:[0-9]+}/{height:[0-9]+}{extension:(?:\\..*)?}",
		s.wrapPublicHandler(s.handler.ImageRandomHandler))
	s.mux.HandleFunc("/image/id/{id}/{width:[0-9]+}/{height:[0-9]+}{extension:(?:\\..*)?}",
		s.wrapPublicHandler(s.handler.ImageResizeHandler))

	s.mux.HandleFunc("/api/scs", s.wrapPublicHandler(s.handler.ScsHandler))
	s.mux.HandleFunc("/api/sc", s.wrapPublicHandler(s.handler.ScHandler))
	s.mux.HandleFunc("/api/sc/search", s.wrapPublicHandler(s.handler.SearchContentHandler))
	s.mux.HandleFunc("/api/sc/tags", s.wrapPublicHandler(s.handler.ContentsTagsHandler))
	s.mux.HandleFunc("/api/sc/hash", s.wrapPublicHandler(s.handler.ScHashHandler))

	s.mux.HandleFunc("/api/themes", s.wrapPublicHandler(s.handler.ThemesHandler))
	s.mux.HandleFunc("/api/theme", s.wrapPublicHandler(s.handler.ThemeHandler))
	s.mux.HandleFunc("/api/theme/search", s.wrapPublicHandler(s.handler.SearchContentHandler))
	s.mux.HandleFunc("/api/theme/tags", s.wrapPublicHandler(s.handler.ContentsTagsHandler))
	s.mux.HandleFunc("/api/theme/hash", s.wrapPublicHandler(s.handler.ThemeHashHandler))

	s.mux.HandleFunc("/api/search", s.wrapContentHandler(s.handler.SearchContentHandler))
	s.mux.HandleFunc("/api/search2", s.wrapContentHandler(s.handler.SearchContentHandler2))

	s.mux.HandleFunc("/api/preview", s.wrapContentHandler(s.handler.PreviewContentHandler))
	s.mux.HandleFunc("/api/build", s.wrapContentHandler(s.handler.BuildContentHandler))
	s.mux.HandleFunc("/api/deploy", s.wrapContentHandler(s.handler.DeployContentHandler))
	s.mux.HandleFunc("/api/deploy/progress", s.handler.DeployProgressHandler)

}

func (s *Server) wrapContentHandler(handler http.HandlerFunc) http.HandlerFunc {
	return s.record.Collect(
		s.cors.Handle(
			s.comp.Gzip(
				s.db.Open(
					s.auth.Check(handler)))))
}

// wrapLicenseHandler wraps handlers that require license expiry check
// Returns 403 Forbidden with JSON error if license is expired
func (s *Server) wrapLicenseHandler(handler http.HandlerFunc) http.HandlerFunc {
	return s.record.Collect(
		s.cors.Handle(
			s.comp.Gzip(
				s.db.Open(
					s.auth.Check(
						s.license.CheckExpiry(handler))))))
}

func (s *Server) wrapSignatureHandler(handler http.HandlerFunc) http.HandlerFunc {
	return s.record.Collect(
		s.cors.Handle(
			s.comp.Gzip(
				s.db.OpenFromSignature(
					s.auth.CheckSignature(handler)))))
}

func (s *Server) wrapPublicHandler(handler http.HandlerFunc) http.HandlerFunc {
	return s.record.Collect(s.cors.Handle(s.auth.CheckGetMethod(handler)))
}

// wrapLicenseAuthHandler wraps handlers that require authentication (TOKEN)
// 支持 JSON Content-Type
func (s *Server) wrapLicenseAuthHandler(handler http.HandlerFunc) http.HandlerFunc {
	return s.record.Collect(s.cors.Handle(s.auth.Check(handler)))
}

func (s *Server) wrapCounterHandler(handler http.HandlerFunc) http.HandlerFunc {
	return s.record.Collect(s.cors.Handle(handler))
}

func (s *Server) wrapPreviewHandler(handler http.HandlerFunc) http.HandlerFunc {
	return s.record.Collect(s.cors.Handle(s.db.OpenPublic(s.auth.CheckPostMethod(handler))))
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

	s.mux.PathPrefix("/admin/static/").Handler(http.StripPrefix("/admin/static/", http.FileServer(adminStaticDir())))

	uploadsDir := application.UploadDir()
	s.mux.PathPrefix("/api/uploads/").Handler(s.record.Collect(s.cors.Handle(s.cache.Control(
		http.StripPrefix("/api/uploads/",
			http.FileServer(restrict(http.Dir(uploadsDir))))))))

	previewPath := fmt.Sprintf("/%s/", application.PreviewFolder())
	s.mux.PathPrefix(previewPath).Handler(s.record.Collect(s.cors.Handle(s.cache.Control(
		http.StripPrefix(previewPath,
			http.FileServer(restrict(http.Dir(application.PreviewDir()))))))))

}
